package main

import (
	"C"
	"encoding/json"

	"github.com/status-im/go-waku/waku/v2/protocol/pb"
	"github.com/status-im/go-waku/waku/v2/protocol/store"
)
import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
)

type StorePagingOptions struct {
	PageSize uint64    `json:"pageSize,omitempty"`
	Cursor   *pb.Index `json:"cursor,omitempty"`
	Forward  bool      `json:"forward,omitempty"`
}

type StoreMessagesArgs struct {
	Topic          string             `json:"pubsubTopic,omitempty"`
	ContentFilters []string           `json:"contentFilters,omitempty"`
	StartTime      int64              `json:"startTime,omitempty"`
	EndTime        int64              `json:"endTime,omitempty"`
	PagingOptions  StorePagingOptions `json:"pagingOptions,omitempty"`
}

type StoreMessagesReply struct {
	Messages   []*pb.WakuMessage  `json:"messages,omitempty"`
	PagingInfo StorePagingOptions `json:"pagingInfo,omitempty"`
	Error      string             `json:"error,omitempty"`
}

//export waku_store_query
// Query historic messages using waku store protocol.
// queryJSON must contain a valid json string with the following format:
// {
// 	"pubsubTopic": "...", // optional string
//  "startTime": 1234, // optional, unix epoch time in nanoseconds
//  "endTime": 1234, // optional, unix epoch time in nanoseconds
//  "contentFilters": [ // optional
// 	    {
//	       "contentTopic": "..."
//      }, ...
//  ],
//  "pagingOptions": {// optional pagination information
//      "pageSize": 40, // number
// 		"cursor": { // optional
//			"digest": ...,
//			"receiverTime": ...,
//			"senderTime": ...,
//      }
//		"forward": true, // sort order
//  }
// }
// peerID should contain the ID of a peer supporting the lightpush protocol. Use NULL to automatically select a node
// If ms is greater than 0, the broadcast of the message must happen before the timeout
// (in milliseconds) is reached, or an error will be returned
func waku_store_query(queryJSON *C.char, peerID *C.char, ms C.int) *C.char {
	if wakuNode == nil {
		return makeJSONResponse(ErrWakuNodeNotReady)
	}

	var args StoreMessagesArgs
	err := json.Unmarshal([]byte(C.GoString(queryJSON)), &args)
	if err != nil {
		return makeJSONResponse(err)
	}

	options := []store.HistoryRequestOption{
		store.WithAutomaticRequestId(),
		store.WithPaging(args.PagingOptions.Forward, args.PagingOptions.PageSize),
		store.WithCursor(args.PagingOptions.Cursor),
	}

	pid := C.GoString(peerID)
	if pid != "" {
		p, err := peer.Decode(pid)
		if err != nil {
			return makeJSONResponse(err)
		}
		options = append(options, store.WithPeer(p))
	} else {
		options = append(options, store.WithAutomaticPeerSelection())
	}

	reply := StoreMessagesReply{}

	var ctx context.Context
	var cancel context.CancelFunc

	if ms > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(int(ms))*time.Millisecond)
		defer cancel()
	} else {
		ctx = context.Background()
	}

	res, err := wakuNode.Store().Query(
		ctx,
		store.Query{
			Topic:         args.Topic,
			ContentTopics: args.ContentFilters,
			StartTime:     args.StartTime,
			EndTime:       args.EndTime,
		},
		options...,
	)

	if err != nil {
		reply.Error = err.Error()
		return prepareJSONResponse(reply, nil)
	}
	reply.Messages = res.Messages
	reply.PagingInfo = StorePagingOptions{
		PageSize: args.PagingOptions.PageSize,
		Cursor:   res.Cursor(),
		Forward:  args.PagingOptions.Forward,
	}

	return prepareJSONResponse(reply, nil)
}
