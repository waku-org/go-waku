package rpc

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/status-im/go-waku/waku/v2/node"
	"github.com/status-im/go-waku/waku/v2/protocol"
	"github.com/status-im/go-waku/waku/v2/protocol/pb"
	"go.uber.org/zap"
)

type RelayService struct {
	node *node.WakuNode

	log *zap.Logger

	messages      map[string][]*pb.WakuMessage
	messagesMutex sync.RWMutex

	runner *runnerService
}

type RelayMessageArgs struct {
	Topic   string         `json:"topic,omitempty"`
	Message pb.WakuMessage `json:"message,omitempty"`
}

type TopicsArgs struct {
	Topics []string `json:"topics,omitempty"`
}

type TopicArgs struct {
	Topic string `json:"topic,omitempty"`
}

func NewRelayService(node *node.WakuNode, log *zap.Logger) *RelayService {
	s := &RelayService{
		node:     node,
		log:      log.Named("relay"),
		messages: make(map[string][]*pb.WakuMessage),
	}

	s.runner = newRunnerService(node.Broadcaster(), s.addEnvelope)
	return s
}

func (r *RelayService) addEnvelope(envelope *protocol.Envelope) {
	r.messagesMutex.Lock()
	defer r.messagesMutex.Unlock()

	if _, ok := r.messages[envelope.PubsubTopic()]; !ok {
		return
	}

	r.messages[envelope.PubsubTopic()] = append(r.messages[envelope.PubsubTopic()], envelope.Message())
}

func (r *RelayService) Start() {
	r.runner.Start()
}

func (r *RelayService) Stop() {
	r.runner.Stop()
}

func (r *RelayService) PostV1Message(req *http.Request, args *RelayMessageArgs, reply *SuccessReply) error {
	var err error
	if args.Topic == "" {
		_, err = r.node.Relay().Publish(req.Context(), &args.Message)
	} else {
		_, err = r.node.Relay().PublishToTopic(req.Context(), &args.Message, args.Topic)
	}
	if err != nil {
		r.log.Error("publishing message", zap.Error(err))
		reply.Success = false
		reply.Error = err.Error()
	} else {
		reply.Success = true
	}
	return nil
}

func (r *RelayService) PostV1Subscription(req *http.Request, args *TopicsArgs, reply *SuccessReply) error {
	ctx := req.Context()
	for _, topic := range args.Topics {
		var err error
		if topic == "" {
			_, err = r.node.Relay().Subscribe(ctx)

		} else {
			_, err = r.node.Relay().SubscribeToTopic(ctx, topic)
		}
		if err != nil {
			r.log.Error("subscribing to topic", zap.String("topic", topic), zap.Error(err))
			reply.Success = false
			reply.Error = err.Error()
			return nil
		}
		r.messages[topic] = make([]*pb.WakuMessage, 0)
	}
	reply.Success = true
	return nil
}

func (r *RelayService) DeleteV1Subscription(req *http.Request, args *TopicsArgs, reply *SuccessReply) error {
	ctx := req.Context()
	for _, topic := range args.Topics {
		err := r.node.Relay().Unsubscribe(ctx, topic)
		if err != nil {
			r.log.Error("unsubscribing from topic", zap.String("topic", topic), zap.Error(err))
			reply.Success = false
			reply.Error = err.Error()
			return nil
		}

		delete(r.messages, topic)
	}
	reply.Success = true
	return nil
}

func (r *RelayService) GetV1Messages(req *http.Request, args *TopicArgs, reply *MessagesReply) error {
	r.messagesMutex.Lock()
	defer r.messagesMutex.Unlock()

	if _, ok := r.messages[args.Topic]; !ok {
		return fmt.Errorf("topic %s not subscribed", args.Topic)
	}

	reply.Messages = r.messages[args.Topic]
	r.messages[args.Topic] = make([]*pb.WakuMessage, 0)
	return nil
}
