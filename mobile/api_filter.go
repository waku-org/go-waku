package gowaku

import (
	"context"
	"encoding/json"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/waku-org/go-waku/waku/v2/protocol/filter"
	"github.com/waku-org/go-waku/waku/v2/protocol/filter/pb"
)

type FilterArgument struct {
	Topic          string                            `json:"pubsubTopic,omitempty"`
	ContentFilters []*pb.FilterRequest_ContentFilter `json:"contentFilters,omitempty"`
}

func toContentFilter(filterJSON string) (filter.ContentFilter, error) {
	var f FilterArgument
	err := json.Unmarshal([]byte(filterJSON), &f)
	if err != nil {
		return filter.ContentFilter{}, err
	}

	result := filter.ContentFilter{
		Topic: f.Topic,
	}
	for _, cf := range f.ContentFilters {
		result.ContentTopics = append(result.ContentTopics, cf.ContentTopic)
	}

	return result, err
}

func FilterSubscribe(filterJSON string, peerID string, ms int) string {
	cf, err := toContentFilter(filterJSON)
	if err != nil {
		return MakeJSONResponse(err)
	}

	if wakuState.node == nil {
		return MakeJSONResponse(errWakuNodeNotReady)
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
			return MakeJSONResponse(err)
		}
		fOptions = append(fOptions, filter.WithPeer(p))
	} else {
		fOptions = append(fOptions, filter.WithAutomaticPeerSelection())
	}

	_, f, err := wakuState.node.Filter().Subscribe(ctx, cf, fOptions...)
	if err != nil {
		return MakeJSONResponse(err)
	}

	go func(f filter.Filter) {
		for envelope := range f.Chan {
			send("message", toSubscriptionMessage(envelope))
		}
	}(f)

	return PrepareJSONResponse(true, nil)
}

func FilterUnsubscribe(filterJSON string, ms int) string {
	cf, err := toContentFilter(filterJSON)
	if err != nil {
		return MakeJSONResponse(err)
	}

	if wakuState.node == nil {
		return MakeJSONResponse(errWakuNodeNotReady)
	}

	var ctx context.Context
	var cancel context.CancelFunc

	if ms > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(int(ms))*time.Millisecond)
		defer cancel()
	} else {
		ctx = context.Background()
	}

	err = wakuState.node.Filter().UnsubscribeFilter(ctx, cf)
	if err != nil {
		return MakeJSONResponse(err)
	}

	return MakeJSONResponse(nil)
}
