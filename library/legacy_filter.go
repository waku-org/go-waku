package library

import (
	"context"
	"encoding/json"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/waku-org/go-waku/waku/v2/protocol/legacy_filter"
	"github.com/waku-org/go-waku/waku/v2/protocol/legacy_filter/pb"
)

type LegacyFilterArgument struct {
	Topic          string                            `json:"pubsubTopic,omitempty"`
	ContentFilters []*pb.FilterRequest_ContentFilter `json:"contentFilters,omitempty"`
}

func toLegacyContentFilter(filterJSON string) (legacy_filter.ContentFilter, error) {
	var f LegacyFilterArgument
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

// Deprecated: Use FilterSubscribe instead
func LegacyFilterSubscribe(filterJSON string, peerID string, ms int) error {
	cf, err := toLegacyContentFilter(filterJSON)
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

	_, f, err := wakuState.node.LegacyFilter().Subscribe(ctx, cf, fOptions...)
	if err != nil {
		return err
	}

	go func(f legacy_filter.Filter) {
		for envelope := range f.Chan {
			send("message", toSubscriptionMessage(envelope))
		}
	}(f)

	return nil
}

// Deprecated: Use FilterUnsubscribe or FilterUnsubscribeAll instead
func LegacyFilterUnsubscribe(filterJSON string, ms int) error {
	cf, err := toLegacyContentFilter(filterJSON)
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

	return wakuState.node.LegacyFilter().UnsubscribeFilter(ctx, cf)
}
