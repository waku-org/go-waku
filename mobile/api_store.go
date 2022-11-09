package gowaku

import (
	"C"
	"encoding/json"

	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/store"
)
import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

type storePagingOptions struct {
	PageSize uint64    `json:"pageSize,omitempty"`
	Cursor   *pb.Index `json:"cursor,omitempty"`
	Forward  bool      `json:"forward,omitempty"`
}

type storeMessagesArgs struct {
	Topic          string             `json:"pubsubTopic,omitempty"`
	ContentFilters []pb.ContentFilter `json:"contentFilters,omitempty"`
	StartTime      int64              `json:"startTime,omitempty"`
	EndTime        int64              `json:"endTime,omitempty"`
	PagingOptions  storePagingOptions `json:"pagingOptions,omitempty"`
}

type storeMessagesReply struct {
	Messages   []*pb.WakuMessage  `json:"messages,omitempty"`
	PagingInfo storePagingOptions `json:"pagingInfo,omitempty"`
	Error      string             `json:"error,omitempty"`
}

func StoreQuery(queryJSON string, peerID string, ms int) string {
	if wakuNode == nil {
		return MakeJSONResponse(errWakuNodeNotReady)
	}

	var args storeMessagesArgs
	err := json.Unmarshal([]byte(queryJSON), &args)
	if err != nil {
		return MakeJSONResponse(err)
	}

	options := []store.HistoryRequestOption{
		store.WithAutomaticRequestId(),
		store.WithPaging(args.PagingOptions.Forward, args.PagingOptions.PageSize),
		store.WithCursor(args.PagingOptions.Cursor),
	}

	if peerID != "" {
		p, err := peer.Decode(peerID)
		if err != nil {
			return MakeJSONResponse(err)
		}
		options = append(options, store.WithPeer(p))
	} else {
		options = append(options, store.WithAutomaticPeerSelection())
	}

	reply := storeMessagesReply{}

	var ctx context.Context
	var cancel context.CancelFunc

	if ms > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(int(ms))*time.Millisecond)
		defer cancel()
	} else {
		ctx = context.Background()
	}

	var contentTopics []string
	for _, ct := range args.ContentFilters {
		contentTopics = append(contentTopics, ct.ContentTopic)
	}

	res, err := wakuNode.Store().Query(
		ctx,
		store.Query{
			Topic:         args.Topic,
			ContentTopics: contentTopics,
			StartTime:     args.StartTime,
			EndTime:       args.EndTime,
		},
		options...,
	)

	if err != nil {
		reply.Error = err.Error()
		return PrepareJSONResponse(reply, nil)
	}
	reply.Messages = res.Messages
	reply.PagingInfo = storePagingOptions{
		PageSize: args.PagingOptions.PageSize,
		Cursor:   res.Cursor(),
		Forward:  args.PagingOptions.Forward,
	}

	return PrepareJSONResponse(reply, nil)
}
