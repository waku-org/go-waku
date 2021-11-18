package rpc

import (
	"fmt"
	"net/http"

	"github.com/status-im/go-waku/waku/v2/node"
	"github.com/status-im/go-waku/waku/v2/protocol/filter"
	"github.com/status-im/go-waku/waku/v2/protocol/pb"
)

type FilterService struct {
	node *node.WakuNode
}

type FilterContentFilterArgs struct {
	Topic          string             `json:"topic,omitempty"`
	ContentFilters []pb.ContentFilter `json:"contentFilters,omitempty"`
}

func makeContentFilter(args *FilterContentFilterArgs) filter.ContentFilter {
	var contentTopics []string
	for _, contentFilter := range args.ContentFilters {
		contentTopics = append(contentTopics, contentFilter.ContentTopic)
	}

	return filter.ContentFilter{
		Topic:         args.Topic,
		ContentTopics: contentTopics,
	}
}

func (f *FilterService) PostV1Subscription(req *http.Request, args *FilterContentFilterArgs, reply *SuccessReply) error {
	_, _, err := f.node.Filter().Subscribe(
		req.Context(),
		makeContentFilter(args),
		filter.WithAutomaticPeerSelection(),
	)
	if err != nil {
		log.Error("Error subscribing to topic:", args.Topic, "err:", err)
		reply.Success = false
		reply.Error = err.Error()
		return nil
	}
	reply.Success = true
	return nil
}

func (f *FilterService) DeleteV1Subscription(req *http.Request, args *FilterContentFilterArgs, reply *SuccessReply) error {
	err := f.node.Filter().UnsubscribeFilter(
		req.Context(),
		makeContentFilter(args),
	)
	if err != nil {
		log.Error("Error unsubscribing to topic:", args.Topic, "err:", err)
		reply.Success = false
		reply.Error = err.Error()
		return nil
	}
	reply.Success = true
	return nil
}

func (f *FilterService) GetV1Messages(req *http.Request, args *Empty, reply *Empty) error {
	return fmt.Errorf("not implemented")
}
