package rpc

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"go.uber.org/zap"
)

type RelayService struct {
	node *node.WakuNode

	log *zap.Logger

	messages      map[string][]*pb.WakuMessage
	cacheCapacity int
	messagesMutex sync.RWMutex

	runner *runnerService
}

type RelayMessageArgs struct {
	Topic   string          `json:"topic,omitempty"`
	Message *pb.WakuMessage `json:"message,omitempty"`
}

type TopicsArgs struct {
	Topics []string `json:"topics,omitempty"`
}

type TopicArgs struct {
	Topic string `json:"topic,omitempty"`
}

func NewRelayService(node *node.WakuNode, cacheCapacity int, log *zap.Logger) *RelayService {
	s := &RelayService{
		node:          node,
		cacheCapacity: cacheCapacity,
		log:           log.Named("relay"),
		messages:      make(map[string][]*pb.WakuMessage),
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

	// Keep a specific max number of messages per topic
	if len(r.messages[envelope.PubsubTopic()]) >= r.cacheCapacity {
		r.messages[envelope.PubsubTopic()] = r.messages[envelope.PubsubTopic()][1:]
	}

	r.messages[envelope.PubsubTopic()] = append(r.messages[envelope.PubsubTopic()], envelope.Message())
}

func (r *RelayService) Start() {
	r.messagesMutex.Lock()
	// Node may already be subscribed to some topics when Relay API handlers are installed. Let's add these
	for _, topic := range r.node.Relay().Topics() {
		r.log.Info("adding topic handler for existing subscription", zap.String("topic", topic))
		r.messages[topic] = make([]*pb.WakuMessage, 0)
	}
	r.messagesMutex.Unlock()

	r.runner.Start()
}

func (r *RelayService) Stop() {
	r.runner.Stop()
}

func (r *RelayService) PostV1Message(req *http.Request, args *RelayMessageArgs, reply *SuccessReply) error {
	var err error

	if args.Topic == "" {
		_, err = r.node.Relay().Publish(req.Context(), args.Message)
	} else {
		_, err = r.node.Relay().PublishToTopic(req.Context(), args.Message, args.Topic)
	}
	if err != nil {
		r.log.Error("publishing message", zap.Error(err))
		return err
	}

	*reply = true
	return nil
}

func (r *RelayService) PostV1Subscription(req *http.Request, args *TopicsArgs, reply *SuccessReply) error {
	ctx := req.Context()
	for _, topic := range args.Topics {
		var err error
		if topic == "" {
			var sub *relay.Subscription
			sub, err = r.node.Relay().Subscribe(ctx)
			r.node.Broadcaster().Unregister(&relay.DefaultWakuTopic, sub.C)
		} else {
			var sub *relay.Subscription
			sub, err = r.node.Relay().SubscribeToTopic(ctx, topic)
			r.node.Broadcaster().Unregister(&topic, sub.C)
		}
		if err != nil {
			r.log.Error("subscribing to topic", zap.String("topic", topic), zap.Error(err))
			return err
		}
		r.messagesMutex.Lock()
		r.messages[topic] = make([]*pb.WakuMessage, 0)
		r.messagesMutex.Unlock()
	}

	*reply = true
	return nil
}

func (r *RelayService) DeleteV1Subscription(req *http.Request, args *TopicsArgs, reply *SuccessReply) error {
	ctx := req.Context()
	for _, topic := range args.Topics {
		err := r.node.Relay().Unsubscribe(ctx, topic)
		if err != nil {
			r.log.Error("unsubscribing from topic", zap.String("topic", topic), zap.Error(err))
			return err
		}

		delete(r.messages, topic)
	}

	*reply = true
	return nil
}

func (r *RelayService) GetV1Messages(req *http.Request, args *TopicArgs, reply *RelayMessagesReply) error {
	r.messagesMutex.Lock()
	defer r.messagesMutex.Unlock()

	if _, ok := r.messages[args.Topic]; !ok {
		return fmt.Errorf("topic %s not subscribed", args.Topic)
	}

	*reply = r.messages[args.Topic]

	r.messages[args.Topic] = make([]*pb.WakuMessage, 0)

	return nil
}
