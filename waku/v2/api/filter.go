package api

import (
	"context"
	"sync"
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
	sync.RWMutex
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

	err := sub.subscribe(contentFilter, sub.Config.MaxPeers)

	if err == nil {
		go sub.healthCheckLoop()
		return sub, nil
	} else {
		return nil, err
	}
}

func (apiSub *Sub) Unsubscribe() error {
	apiSub.RLock()
	defer apiSub.RUnlock()
	for _, s := range apiSub.subs {
		apiSub.wf.UnsubscribeWithSubscription(apiSub.ctx, s)
	}
	apiSub.cancel()
	close(apiSub.DataCh)
	return nil
}

func (apiSub *Sub) healthCheckLoop() {
	// Health checks
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-apiSub.ctx.Done():
			return
		case <-ticker.C:
			// Returns a map of pubsub topics to peer counts
			m := apiSub.checkAliveness()
			for t, cnt := range m {
				if cnt < apiSub.Config.MaxPeers {
					cFilter := protocol.ContentFilter{t, apiSub.ContentFilter.ContentTopics}
					apiSub.subscribe(cFilter, apiSub.Config.MaxPeers-cnt)
				}
			}
		}
	}
}

func (apiSub *Sub) checkAliveness() map[string]int {
	apiSub.RLock()
	defer apiSub.RUnlock()

	// Only healthy topics will be pushed here
	ch := make(chan string)

	wg := &sync.WaitGroup{}
	wg.Add(len(apiSub.subs))
	for _, subDetails := range apiSub.subs {
		go func(subDetails *subscription.SubscriptionDetails) {
			defer wg.Done()
			err := apiSub.wf.IsSubscriptionAlive(apiSub.ctx, subDetails)

			if err != nil {
				subDetails.Close()
				apiSub.Lock()
				defer apiSub.Unlock()
				delete(apiSub.subs, subDetails.ID)
			} else {
				ch <- subDetails.ContentFilter.PubsubTopic
			}
		}(subDetails)

	}
	wg.Wait()
	close(ch)
	// Collect healthy topics
	m := make(map[string]int)
	topicMap, _ := protocol.ContentFilterToPubSubTopicMap(apiSub.ContentFilter)
	for _, t := range maps.Keys(topicMap) {
		m[t] = 0
	}
	for t := range ch {
		m[t]++
	}

	return m

}
func (apiSub *Sub) subscribe(contentFilter protocol.ContentFilter, peerCount int) error {
	// Low-level subscribe, returns a set of SubscriptionDetails
	options := make([]filter.FilterSubscribeOption, 0)
	options = append(options, filter.WithMaxPeersPerContentFilter(int(peerCount)))
	for _, p := range apiSub.Config.Peers {
		options = append(options, filter.WithPeer(p))
	}
	subs, err := apiSub.wf.Subscribe(apiSub.ctx, contentFilter, options...)

	if err != nil {
		// TODO what if fails?
		return err
	}
	apiSub.Lock()
	defer apiSub.Unlock()
	for _, s := range subs {
		apiSub.subs[s.ID] = s
	}
	// Multiplex onto single channel
	// Goroutines will exit once sub channels are closed
	for _, subDetails := range subs {
		go func(subDetails *subscription.SubscriptionDetails) {
			apiSub.log.Info("New multiplex", zap.String("sub ID", subDetails.ID))
			for env := range subDetails.C {
				apiSub.DataCh <- env
			}
		}(subDetails)
	}
	return nil

}
