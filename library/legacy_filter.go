package library

import (
	"context"
	"encoding/json"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/waku-org/go-waku/waku/v2/protocol/legacy_filter"
	"github.com/waku-org/go-waku/waku/v2/protocol/legacy_filter/pb"
)

type legacyFilterArgument struct {
	Topic          string                            `json:"pubsubTopic,omitempty"`
	ContentFilters []*pb.FilterRequest_ContentFilter `json:"contentFilters,omitempty"`
}

func toLegacyContentFilter(filterJSON string) (legacy_filter.ContentFilter, error) {
	var f legacyFilterArgument
	err := json.Unmarshal([]byte(filterJSON), &f)
	if err != nil {
		return legacy_filter.ContentFilter{}, err
	}

	result := legacy_filter.ContentFilter{
		Topic: f.Topic,
	}
	for _, cf := range f.ContentFilters {
		result.ContentTopics = append(result.ContentTopics, cf.ContentTopic)
	}

	return result, err
}

// LegacyFilterSubscribe is used to create a subscription to a filter node to receive messages
// Deprecated: Use FilterSubscribe instead
func LegacyFilterSubscribe(instance *WakuInstance, filterJSON string, peerID string, ms int) error {
	cf, err := toLegacyContentFilter(filterJSON)
	if err != nil {
		return err
	}

	if err := validateInstance(instance, MustBeStarted); err != nil {
		return err
	}

	var ctx context.Context
	var cancel context.CancelFunc

	if ms > 0 {
		ctx, cancel = context.WithTimeout(instance.ctx, time.Duration(int(ms))*time.Millisecond)
		defer cancel()
	} else {
		ctx = instance.ctx
	}

	var fOptions []legacy_filter.FilterSubscribeOption
	if peerID != "" {
		p, err := peer.Decode(peerID)
		if err != nil {
			return err
		}
		fOptions = append(fOptions, legacy_filter.WithPeer(p))
	} else {
		fOptions = append(fOptions, legacy_filter.WithAutomaticPeerSelection())
	}

	_, f, err := instance.node.LegacyFilter().Subscribe(ctx, cf, fOptions...)
	if err != nil {
		return err
	}

	go func(f legacy_filter.Filter) {
		for envelope := range f.Chan {
			send(instance, "message", toSubscriptionMessage(envelope))
		}
	}(f)

	return nil
}

// LegacyFilterUnsubscribe is used to remove a filter criteria from an active subscription with a filter node
// Deprecated: Use FilterUnsubscribe or FilterUnsubscribeAll instead
func LegacyFilterUnsubscribe(instance *WakuInstance, filterJSON string, ms int) error {
	cf, err := toLegacyContentFilter(filterJSON)
	if err != nil {
		return err
	}

	if err := validateInstance(instance, MustBeStarted); err != nil {
		return err
	}

	var ctx context.Context
	var cancel context.CancelFunc

	if ms > 0 {
		ctx, cancel = context.WithTimeout(instance.ctx, time.Duration(int(ms))*time.Millisecond)
		defer cancel()
	} else {
		ctx = instance.ctx
	}

	return instance.node.LegacyFilter().UnsubscribeFilter(ctx, cf)
}
