package library

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/waku-org/go-waku/waku/v2/protocol/filter"
)

type filterArgument struct {
	Topic         string   `json:"pubsubTopic,omitempty"`
	ContentTopics []string `json:"contentTopics,omitempty"`
}

func toContentFilter(filterJSON string) (filter.ContentFilter, error) {
	var f filterArgument
	err := json.Unmarshal([]byte(filterJSON), &f)
	if err != nil {
		return filter.ContentFilter{}, err
	}

	return filter.ContentFilter{
		Topic:         f.Topic,
		ContentTopics: f.ContentTopics,
	}, nil
}

type subscribeResult struct {
	Subscriptions []*filter.SubscriptionDetails `json:"subscriptions"`
	Error         string                        `json:"error,omitempty"`
}

// FilterSubscribe is used to create a subscription to a filter node to receive messages
func FilterSubscribe(filterJSON string, peerID string, ms int) (string, error) {
	cf, err := toContentFilter(filterJSON)
	if err != nil {
		return "", err
	}

	if wakuState.node == nil {
		return "", errWakuNodeNotReady
	}

	var ctx context.Context
	var cancel context.CancelFunc

	if ms > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(int(ms))*time.Millisecond)
		defer cancel()
	} else {
		ctx = context.Background()
	}

	var fOptions []filter.FilterSubscribeOption
	if peerID != "" {
		p, err := peer.Decode(peerID)
		if err != nil {
			return "", err
		}
		fOptions = append(fOptions, filter.WithPeer(p))
	} else {
		fOptions = append(fOptions, filter.WithAutomaticPeerSelection())
	}

	subscriptions, err := wakuState.node.FilterLightnode().Subscribe(ctx, cf, fOptions...)
	if err != nil && subscriptions == nil {
		return "", err
	}

	for _, subscriptionDetails := range subscriptions {
		go func(subscriptionDetails *filter.SubscriptionDetails) {
			for envelope := range subscriptionDetails.C {
				send("message", toSubscriptionMessage(envelope))
			}
		}(subscriptionDetails)
	}
	var subResult subscribeResult
	subResult.Subscriptions = subscriptions
	subResult.Error = err.Error()
	return marshalJSON(subResult)
}

// FilterPing is used to determine if a peer has an active subscription
func FilterPing(peerID string, ms int) error {
	if wakuState.node == nil {
		return errWakuNodeNotReady
	}

	var ctx context.Context
	var cancel context.CancelFunc

	if ms > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(int(ms))*time.Millisecond)
		defer cancel()
	} else {
		ctx = context.Background()
	}

	var pID peer.ID
	var err error
	if peerID != "" {
		pID, err = peer.Decode(peerID)
		if err != nil {
			return err
		}
	} else {
		return errors.New("peerID is required")
	}

	return wakuState.node.FilterLightnode().Ping(ctx, pID)
}

// FilterUnsubscribe is used to remove a filter criteria from an active subscription with a filter node
func FilterUnsubscribe(filterJSON string, peerID string, ms int) error {
	cf, err := toContentFilter(filterJSON)
	if err != nil {
		return err
	}

	if wakuState.node == nil {
		return errWakuNodeNotReady
	}

	var ctx context.Context
	var cancel context.CancelFunc

	if ms > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(int(ms))*time.Millisecond)
		defer cancel()
	} else {
		ctx = context.Background()
	}

	var fOptions []filter.FilterUnsubscribeOption
	if peerID != "" {
		p, err := peer.Decode(peerID)
		if err != nil {
			return err
		}
		fOptions = append(fOptions, filter.Peer(p))
	} else {
		return errors.New("peerID is required")
	}

	pushResult, err := wakuState.node.FilterLightnode().Unsubscribe(ctx, cf, fOptions...)
	if err != nil {
		return err
	}

	result := <-pushResult

	return result.Err
}

type unsubscribeAllResult struct {
	PeerID string `json:"peerID"`
	Error  string `json:"error"`
}

// FilterUnsubscribeAll is used to remove an active subscription to a peer. If no peerID is defined, it will stop all active filter subscriptions
func FilterUnsubscribeAll(peerID string, ms int) (string, error) {
	if wakuState.node == nil {
		return "", errWakuNodeNotReady
	}

	var ctx context.Context
	var cancel context.CancelFunc

	if ms > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(int(ms))*time.Millisecond)
		defer cancel()
	} else {
		ctx = context.Background()
	}

	var fOptions []filter.FilterUnsubscribeOption
	if peerID != "" {
		p, err := peer.Decode(peerID)
		if err != nil {
			return "", err
		}
		fOptions = append(fOptions, filter.Peer(p))
	} else {
		fOptions = append(fOptions, filter.UnsubscribeAll())
	}

	pushResult, err := wakuState.node.FilterLightnode().UnsubscribeAll(ctx, fOptions...)
	if err != nil {
		return "", err
	}

	var unsubscribeResult []unsubscribeAllResult

	for result := range pushResult {
		ur := unsubscribeAllResult{
			PeerID: result.PeerID.Pretty(),
		}
		if result.Err != nil {
			ur.Error = result.Err.Error()
		}
		unsubscribeResult = append(unsubscribeResult, ur)
	}

	return marshalJSON(unsubscribeResult)
}
