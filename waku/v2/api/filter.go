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

func Subscribe(ctx context.Context, wf *filter.WakuFilterLightNode, contentFilter protocol.ContentFilter, config FilterConfig) (*Sub, error) {
	sub := new(Sub)
	sub.wf = wf
	sub.ctx, sub.cancel = context.WithCancel(ctx)
	sub.subs = make(subscription.SubscriptionSet)
	sub.DataCh = make(chan *protocol.Envelope)
	sub.ContentFilter = contentFilter
	sub.Config = config
	sub.log = func() *zap.Logger { log, _ := zap.NewDevelopment(); return log }().Named("filterv2-api")

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
	for _, s := range apiSub.subs {
		apiSub.wf.UnsubscribeWithSubscription(apiSub.ctx, s)
	}
	close(apiSub.DataCh)

}

func (apiSub *Sub) healthCheckLoop() {
	// Health checks
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-apiSub.ctx.Done():
			apiSub.log.Info("healthCheckLoop: Done()")
			return
		case <-ticker.C:
			apiSub.log.Info("healthCheckLoop: checkAliveness()")
			apiSub.checkAliveness()
		}
	}

}

func (apiSub *Sub) checkAliveness() {
	apiSub.log.Info("ENTER checkAliveness()")

	// Buffered chan for sub aliveness results
	type CheckResult struct {
		sub   *subscription.SubscriptionDetails
		alive bool
	}
	ch := make(chan CheckResult, len(apiSub.subs))

	// Run pings asynchronously
	for _, s := range apiSub.subs {
		go func() {
			ctx, _ := context.WithTimeout(apiSub.ctx, 5*time.Second)
			err := apiSub.wf.IsSubscriptionAlive(ctx, s)

			ch <- CheckResult{s, err == nil}
		}()
	}

	// Collect healthy topic counts
	topicCounts := make(map[string]int)

	topicMap, _ := protocol.ContentFilterToPubSubTopicMap(apiSub.ContentFilter)
	for _, t := range maps.Keys(topicMap) {
		topicCounts[t] = 0
	}
	// Close inactive subs
	cnt := 0
	for s := range ch {
		cnt++
		if !s.alive {
			s.sub.Close()
			delete(apiSub.subs, s.sub.ID)
		} else {
			topicCounts[s.sub.ContentFilter.PubsubTopic]++
		}

		if cnt == len(apiSub.subs) {
			// All values received
			break
		}
	}
	close(ch)
	for t, cnt := range topicCounts {
		if cnt == apiSub.Config.MaxPeers {
			delete(topicCounts, t)
		}
	}
	// Re-subscribe asynchronously
	newSubs := make(chan []*subscription.SubscriptionDetails)
	for t, cnt := range topicCounts {
		cFilter := protocol.ContentFilter{t, apiSub.ContentFilter.ContentTopics}
		go func() {
			subs, err := apiSub.subscribe(cFilter, apiSub.Config.MaxPeers-cnt)
			if err != nil {
				newSubs <- subs
			}
		}()
	}

	for subs := range newSubs {
		apiSub.multiplex(subs)
		if cnt == len(topicCounts) {
			break
		}
	}
	close(newSubs)

	apiSub.log.Info("EXIT checkAliveness()")
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
