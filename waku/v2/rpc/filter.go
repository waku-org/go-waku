package rpc

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/status-im/go-waku/waku/v2/node"
	"github.com/status-im/go-waku/waku/v2/protocol"
	"github.com/status-im/go-waku/waku/v2/protocol/filter"
	"github.com/status-im/go-waku/waku/v2/protocol/pb"
	"go.uber.org/zap"
)

type FilterService struct {
	node *node.WakuNode
	log  *zap.SugaredLogger

	messages      map[string][]*pb.WakuMessage
	messagesMutex sync.RWMutex

	runner *runnerService
}

type FilterContentArgs struct {
	Topic          string             `json:"topic,omitempty"`
	ContentFilters []pb.ContentFilter `json:"contentFilters,omitempty"`
}

type ContentTopicArgs struct {
	ContentTopic string `json:"contentTopic,omitempty"`
}

func NewFilterService(node *node.WakuNode, log *zap.SugaredLogger) *FilterService {
	s := &FilterService{
		node:     node,
		log:      log.Named("filter"),
		messages: make(map[string][]*pb.WakuMessage),
	}
	s.runner = newRunnerService(node.Broadcaster(), s.addEnvelope)
	return s
}

func makeContentFilter(args *FilterContentArgs) filter.ContentFilter {
	var contentTopics []string
	for _, contentFilter := range args.ContentFilters {
		contentTopics = append(contentTopics, contentFilter.ContentTopic)
	}

	return filter.ContentFilter{
		Topic:         args.Topic,
		ContentTopics: contentTopics,
	}
}

func (f *FilterService) addEnvelope(envelope *protocol.Envelope) {
	f.messagesMutex.Lock()
	defer f.messagesMutex.Unlock()

	contentTopic := envelope.Message().ContentTopic
	if _, ok := f.messages[contentTopic]; !ok {
		return
	}

	f.messages[contentTopic] = append(f.messages[contentTopic], envelope.Message())
}

func (f *FilterService) Start() {
	f.runner.Start()
}

func (f *FilterService) Stop() {
	f.runner.Stop()
}

func (f *FilterService) PostV1Subscription(req *http.Request, args *FilterContentArgs, reply *SuccessReply) error {
	_, _, err := f.node.Filter().Subscribe(
		req.Context(),
		makeContentFilter(args),
		filter.WithAutomaticPeerSelection(),
	)
	if err != nil {
		f.log.Error("Error subscribing to topic:", args.Topic, "err:", err)
		reply.Success = false
		reply.Error = err.Error()
		return nil
	}
	for _, contentFilter := range args.ContentFilters {
		f.messages[contentFilter.ContentTopic] = make([]*pb.WakuMessage, 0)
	}
	reply.Success = true
	return nil
}

func (f *FilterService) DeleteV1Subscription(req *http.Request, args *FilterContentArgs, reply *SuccessReply) error {
	err := f.node.Filter().UnsubscribeFilter(
		req.Context(),
		makeContentFilter(args),
	)
	if err != nil {
		f.log.Error("Error unsubscribing to topic:", args.Topic, "err:", err)
		reply.Success = false
		reply.Error = err.Error()
		return nil
	}
	for _, contentFilter := range args.ContentFilters {
		delete(f.messages, contentFilter.ContentTopic)
	}

	reply.Success = true
	return nil
}

func (f *FilterService) GetV1Messages(req *http.Request, args *ContentTopicArgs, reply *MessagesReply) error {
	f.messagesMutex.Lock()
	defer f.messagesMutex.Unlock()

	if _, ok := f.messages[args.ContentTopic]; !ok {
		return fmt.Errorf("topic %s not subscribed", args.ContentTopic)
	}

	reply.Messages = f.messages[args.ContentTopic]
	f.messages[args.ContentTopic] = make([]*pb.WakuMessage, 0)
	return nil
}
