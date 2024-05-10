package api

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/filter"
	"github.com/waku-org/go-waku/waku/v2/protocol/subscription"
	"go.uber.org/zap"
	"golang.org/x/exp/maps"
)

type FilterConfig struct {
	MaxPeers int
	Peers    []peer.ID
}

type Sub struct {
	ContentFilter protocol.ContentFilter
	DataCh        chan *protocol.Envelope
	Config        FilterConfig
	subs          subscription.SubscriptionSet
	wf            *filter.WakuFilterLightNode
	ctx           context.Context
	cancel        context.CancelFunc
	log           *zap.Logger
}

// Subscribe
func Subscribe(ctx context.Context, wf *filter.WakuFilterLightNode, contentFilter protocol.ContentFilter, config FilterConfig) (*Sub, error) {
	sub := new(Sub)
	sub.wf = wf
	sub.ctx, sub.cancel = context.WithCancel(ctx)
	sub.subs = make(subscription.SubscriptionSet)
	sub.DataCh = make(chan *protocol.Envelope)
	sub.ContentFilter = contentFilter
	sub.Config = config
	sub.log = func() *zap.Logger {
		log, _ := zap.NewDevelopment()
		return log
	}().Named("filterv2-api")

	subs, err := sub.subscribe(contentFilter, sub.Config.MaxPeers)

	if err == nil {
		sub.multiplex(subs)
		sub.log.Info("go sub.healthCheckLoop()")
		go sub.healthCheckLoop()
		return sub, nil
	} else {
		return nil, err
	}
}

func (apiSub *Sub) Unsubscribe() {
	apiSub.cancel()

}

func (apiSub *Sub) healthCheckLoop() {
	// Health checks
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-apiSub.ctx.Done():
			apiSub.log.Info("healthCheckLoop: Done()")
			apiSub.cleanup()
			return
		case <-ticker.C:
			apiSub.log.Info("healthCheckLoop: checkAliveness()")
			topicCounts := apiSub.getTopicCounts()
			apiSub.resubscribe(topicCounts)
		}
	}

}

func (apiSub *Sub) cleanup() {
	apiSub.log.Info("ENTER cleanup()")
	defer func() {
		apiSub.log.Info("EXIT cleanup()")
	}()

	for _, s := range apiSub.subs {
		_, err := apiSub.wf.UnsubscribeWithSubscription(apiSub.ctx, s)
		if err != nil {
			//Logging with info as this is part of cleanup
			apiSub.log.Info("failed to unsubscribe filter", zap.Error(err))
		}
	}
	close(apiSub.DataCh)

}

// Returns active sub counts for each pubsub topic
func (apiSub *Sub) getTopicCounts() map[string]int {
	apiSub.log.Info("ENTER getTopicCounts()")
	defer func() {
		apiSub.log.Info("EXIT getTopicCounts()")
	}()

	// Buffered chan for sub aliveness results
	type CheckResult struct {
		sub   *subscription.SubscriptionDetails
		alive bool
	}
	checkResults := make(chan CheckResult, len(apiSub.subs))

	// Run pings asynchronously
	for _, s := range apiSub.subs {
		go func(sub *subscription.SubscriptionDetails) {
			ctx, cancelFunc := context.WithTimeout(apiSub.ctx, 5*time.Second)
			defer cancelFunc()
			err := apiSub.wf.IsSubscriptionAlive(ctx, sub)

			apiSub.log.Info("Check result:", zap.Any("subID", sub.ID), zap.Bool("result", err == nil))
			checkResults <- CheckResult{sub, err == nil}
		}(s)
	}

	// Collect healthy topic counts
	topicCounts := make(map[string]int)

	topicMap, _ := protocol.ContentFilterToPubSubTopicMap(apiSub.ContentFilter)
	for _, t := range maps.Keys(topicMap) {
		topicCounts[t] = 0
	}
	cnt := 0
	subLen := len(apiSub.subs)
	for s := range checkResults {
		cnt++
		if !s.alive {
			// Close inactive subs
			s.sub.Close()
			delete(apiSub.subs, s.sub.ID)
		} else {
			topicCounts[s.sub.ContentFilter.PubsubTopic]++
		}

		if cnt == subLen {
			// All values received
			break
		}
	}

	close(checkResults)
	return topicCounts
}

// Attempts to resubscribe on topics that lack subscriptions
func (apiSub *Sub) resubscribe(topicCounts map[string]int) {
	apiSub.log.Info("ENTER resubscribe()")
	defer func() {
		apiSub.log.Info("EXIT resubscribe()")
	}()

	// Delete healthy topics
	for t, cnt := range topicCounts {
		if cnt == apiSub.Config.MaxPeers {
			delete(topicCounts, t)
		}
	}

	if len(topicCounts) == 0 {
		// All topics healthy, return
		return
	}

	// Re-subscribe asynchronously
	newSubs := make(chan []*subscription.SubscriptionDetails)

	for t, cnt := range topicCounts {
		cFilter := protocol.ContentFilter{PubsubTopic: t, ContentTopics: apiSub.ContentFilter.ContentTopics}
		go func(count int) {
			subs, _ := apiSub.subscribe(cFilter, apiSub.Config.MaxPeers-count)
			newSubs <- subs
		}(cnt)
	}

	cnt := 0
	apiSub.log.Info("resubscribe(): before range newSubs")
	for subs := range newSubs {
		cnt++
		if subs != nil {
			apiSub.multiplex(subs)
		}
		if cnt == len(topicCounts) {
			// Received all subscription results
			break
		}
	}
	apiSub.log.Info("checkAliveness(): close(newSubs)")
	close(newSubs)
}

func (apiSub *Sub) subscribe(contentFilter protocol.ContentFilter, peerCount int) ([]*subscription.SubscriptionDetails, error) {
	// Low-level subscribe, returns a set of SubscriptionDetails
	options := make([]filter.FilterSubscribeOption, 0)
	options = append(options, filter.WithMaxPeersPerContentFilter(int(peerCount)))
	for _, p := range apiSub.Config.Peers {
		options = append(options, filter.WithPeer(p))
	}
	subs, err := apiSub.wf.Subscribe(apiSub.ctx, contentFilter, options...)

	if err != nil {
		// TODO what if fails?
		return nil, err
	}

	return subs, nil
}

func (apiSub *Sub) multiplex(subs []*subscription.SubscriptionDetails) {
	for _, s := range subs {
		apiSub.subs[s.ID] = s
	}
	// Multiplex onto single channel
	// Goroutines will exit once sub channels are closed
	for _, subDetails := range subs {
		go func(subDetails *subscription.SubscriptionDetails) {
			apiSub.log.Info("New multiplex", zap.String("subID", subDetails.ID))
			for env := range subDetails.C {
				apiSub.DataCh <- env
			}
		}(subDetails)
	}
}
