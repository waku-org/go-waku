package library

import (
	"C"
	"encoding/json"

	wpb "github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/store"
	"github.com/waku-org/go-waku/waku/v2/protocol/store/pb"
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
	Topic         string              `json:"pubsubTopic,omitempty"`
	ContentTopics []string            `json:"contentTopics,omitempty"`
	StartTime     *int64              `json:"startTime,omitempty"`
	EndTime       *int64              `json:"endTime,omitempty"`
	PagingOptions *storePagingOptions `json:"pagingOptions,omitempty"`
}

type storeMessagesReply struct {
	Messages   []*wpb.WakuMessage `json:"messages,omitempty"`
	PagingInfo storePagingOptions `json:"pagingInfo,omitempty"`
	Error      string             `json:"error,omitempty"`
}

func queryResponse(ctx context.Context, instance *WakuInstance, args storeMessagesArgs, options []store.HistoryRequestOption) (string, error) {
	res, err := instance.node.LegacyStore().Query(
		ctx,
		store.Query{
			PubsubTopic:   args.Topic,
			ContentTopics: args.ContentTopics,
			StartTime:     args.StartTime,
			EndTime:       args.EndTime,
		},
		options...,
	)

	reply := storeMessagesReply{}

	if err != nil {
		reply.Error = err.Error()
		return marshalJSON(reply)
	}
	reply.Messages = res.Messages
	reply.PagingInfo = storePagingOptions{
		PageSize: args.PagingOptions.PageSize,
		Cursor:   res.Cursor(),
		Forward:  args.PagingOptions.Forward,
	}

	return marshalJSON(reply)
}

// StoreQuery is used to retrieve historic messages using waku store protocol.
func StoreQuery(instance *WakuInstance, queryJSON string, peerID string, ms int) (string, error) {
	if err := validateInstance(instance, MustBeStarted); err != nil {
		return "", err
	}

	var args storeMessagesArgs
	err := json.Unmarshal([]byte(queryJSON), &args)
	if err != nil {
		return "", err
	}

	options := []store.HistoryRequestOption{
		store.WithAutomaticRequestID(),
		store.WithPaging(args.PagingOptions.Forward, args.PagingOptions.PageSize),
		store.WithCursor(args.PagingOptions.Cursor),
	}

	if peerID != "" {
		p, err := peer.Decode(peerID)
		if err != nil {
			return "", err
		}
		options = append(options, store.WithPeer(p))
	} else {
		options = append(options, store.WithAutomaticPeerSelection())
	}

	var ctx context.Context
	var cancel context.CancelFunc

	if ms > 0 {
		ctx, cancel = context.WithTimeout(instance.ctx, time.Duration(int(ms))*time.Millisecond)
		defer cancel()
	} else {
		ctx = instance.ctx
	}

	return queryResponse(ctx, instance, args, options)
}

// StoreLocalQuery is used to retrieve historic messages stored in the localDB using waku store protocol.
func StoreLocalQuery(instance *WakuInstance, queryJSON string) (string, error) {
	if err := validateInstance(instance, MustBeStarted); err != nil {
		return "", err
	}

	var args storeMessagesArgs
	err := json.Unmarshal([]byte(queryJSON), &args)
	if err != nil {
		return "", err
	}

	options := []store.HistoryRequestOption{
		store.WithAutomaticRequestID(),
		store.WithPaging(args.PagingOptions.Forward, args.PagingOptions.PageSize),
		store.WithCursor(args.PagingOptions.Cursor),
		store.WithLocalQuery(),
	}

	return queryResponse(instance.ctx, instance, args, options)
}
