package gowaku

import (
	"context"
	"encoding/json"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/status-im/go-waku/waku/v2/protocol/filter"
	"github.com/status-im/go-waku/waku/v2/protocol/pb"
)

type FilterArgument struct {
	Topic          string             `json:"topic,omitempty"`
	ContentFilters []pb.ContentFilter `json:"contentFilters,omitempty"`
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
		return makeJSONResponse(err)
	}

	if wakuNode == nil {
		return makeJSONResponse(errWakuNodeNotReady)
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
			return makeJSONResponse(err)
		}
		fOptions = append(fOptions, filter.WithPeer(p))
	} else {
		fOptions = append(fOptions, filter.WithAutomaticPeerSelection())
	}

	_, f, err := wakuNode.Filter().Subscribe(ctx, cf, fOptions...)
	if err != nil {
		return makeJSONResponse(err)
	}

	go func(f filter.Filter) {
		for envelope := range f.Chan {
			send("message", toSubscriptionMessage(envelope))
		}
	}(f)

	return prepareJSONResponse(true, nil)
}

func FilterUnsubscribe(filterJSON string, ms int) string {
	cf, err := toContentFilter(filterJSON)
	if err != nil {
		return makeJSONResponse(err)
	}

	if wakuNode == nil {
		return makeJSONResponse(errWakuNodeNotReady)
	}

	var ctx context.Context
	var cancel context.CancelFunc

	if ms > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(int(ms))*time.Millisecond)
		defer cancel()
	} else {
		ctx = context.Background()
	}

	err = wakuNode.Filter().UnsubscribeFilter(ctx, cf)
	if err != nil {
		return makeJSONResponse(err)
	}

	return makeJSONResponse(nil)
}
