package rpc

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/status-im/go-waku/waku/v2/node"
	"github.com/status-im/go-waku/waku/v2/protocol"
	"github.com/status-im/go-waku/waku/v2/protocol/pb"
	"github.com/status-im/go-waku/waku/v2/protocol/relay"
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
	Topic   string              `json:"topic,omitempty"`
	Message RPCWakuRelayMessage `json:"message,omitempty"`
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
	// Node may already be subscribed to some topics when Relay API handlers are installed. Let's add these
	for _, topic := range r.node.Relay().Topics() {
		r.log.Info("adding topic handler for existing subscription", zap.String("topic", topic))
		r.messages[topic] = make([]*pb.WakuMessage, 0)
	}

	r.runner.Start()
}

func (r *RelayService) Stop() {
	r.runner.Stop()
}

func (r *RelayService) PostV1Message(req *http.Request, args *RelayMessageArgs, reply *SuccessReply) error {
	var err error

	msg := args.Message.toProto()

	if args.Topic == "" {
		_, err = r.node.Relay().Publish(req.Context(), msg)
	} else {
		_, err = r.node.Relay().PublishToTopic(req.Context(), msg, args.Topic)
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
		r.messages[topic] = make([]*pb.WakuMessage, 0)
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

	for i := range r.messages[args.Topic] {
		*reply = append(*reply, ProtoWakuMessageToRPCWakuRelayMessage(r.messages[args.Topic][i]))
	}

	r.messages[args.Topic] = make([]*pb.WakuMessage, 0)

	return nil
}
