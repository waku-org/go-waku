package gowaku

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/waku-org/go-waku/waku/v2/protocol/filter"
)

type FilterArgument struct {
	Topic         string   `json:"pubsubTopic,omitempty"`
	ContentTopics []string `json:"contentTopics,omitempty"`
}

func toContentFilter(filterJSON string) (filter.ContentFilter, error) {
	var f FilterArgument
	err := json.Unmarshal([]byte(filterJSON), &f)
	if err != nil {
		return filter.ContentFilter{}, err
	}

	return filter.ContentFilter{
		Topic:         f.Topic,
		ContentTopics: f.ContentTopics,
	}, nil
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

	subscriptionDetails, err := wakuState.node.FilterLightnode().Subscribe(ctx, cf, fOptions...)
	if err != nil {
		return MakeJSONResponse(err)
	}

	go func(subscriptionDetails *filter.SubscriptionDetails) {
		for envelope := range subscriptionDetails.C {
			send("message", toSubscriptionMessage(envelope))
		}
	}(subscriptionDetails)

	return PrepareJSONResponse(subscriptionDetails, nil)
}

func FilterPing(peerID string, ms int) string {
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

	var pID peer.ID
	var err error
	if peerID != "" {
		pID, err = peer.Decode(peerID)
		if err != nil {
			return MakeJSONResponse(err)
		}
	} else {
		return MakeJSONResponse(errors.New("peerID is required"))
	}

	err = wakuState.node.FilterLightnode().Ping(ctx, pID)

	return MakeJSONResponse(err)
}

func FilterUnsubscribe(filterJSON string, peerID string, ms int) string {
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

	var fOptions []filter.FilterUnsubscribeOption
	if peerID != "" {
		p, err := peer.Decode(peerID)
		if err != nil {
			return MakeJSONResponse(err)
		}
		fOptions = append(fOptions, filter.Peer(p))
	} else {
		return MakeJSONResponse(errors.New("peerID is required"))
	}

	pushResult, err := wakuState.node.FilterLightnode().Unsubscribe(ctx, cf, fOptions...)
	if err != nil {
		return MakeJSONResponse(err)
	}

	result := <-pushResult

	return MakeJSONResponse(result.Err)
}

type unsubscribeAllResult struct {
	PeerID string `json:"peerID"`
	Error  string `json:"error"`
}

func FilterUnsubscribeAll(peerID string, ms int) string {
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

	var fOptions []filter.FilterUnsubscribeOption
	if peerID != "" {
		p, err := peer.Decode(peerID)
		if err != nil {
			return MakeJSONResponse(err)
		}
		fOptions = append(fOptions, filter.Peer(p))
	} else {
		fOptions = append(fOptions, filter.UnsubscribeAll())
	}

	pushResult, err := wakuState.node.FilterLightnode().UnsubscribeAll(ctx, fOptions...)
	if err != nil {
		return MakeJSONResponse(err)
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

	return PrepareJSONResponse(unsubscribeResult, nil)
}
