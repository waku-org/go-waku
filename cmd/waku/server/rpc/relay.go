package rpc

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/waku-org/go-waku/cmd/waku/server"
	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"go.uber.org/zap"
)

// RelayService represents the JSON RPC service for WakuRelay
type RelayService struct {
	node *node.WakuNode

	log *zap.Logger

	messages      map[string][]*pb.WakuMessage
	cacheCapacity int
	messagesMutex sync.RWMutex

	runner *runnerService
}

// RelayMessageArgs represents the requests used for posting messages
type RelayMessageArgs struct {
	Topic   string          `json:"topic,omitempty"`
	Message *RPCWakuMessage `json:"message,omitempty"`
}

// RelayAutoMessageArgs represents the requests used for posting messages
type RelayAutoMessageArgs struct {
	Message *RPCWakuMessage `json:"message,omitempty"`
}

// TopicsArgs represents the lists of topics to use when subscribing / unsubscribing
type TopicsArgs struct {
	Topics []string `json:"topics,omitempty"`
}

// TopicArgs represents a request that contains a single topic
type TopicArgs struct {
	Topic string `json:"topic,omitempty"`
}

// NewRelayService returns an instance of RelayService
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

// Start starts the RelayService
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

// Stop stops the RelayService
func (r *RelayService) Stop() {
	r.runner.Stop()
}

// PostV1Message is invoked when the json rpc request uses the post_waku_v2_relay_v1_message method
func (r *RelayService) PostV1Message(req *http.Request, args *RelayMessageArgs, reply *SuccessReply) error {
	var err error

	topic := relay.DefaultWakuTopic
	if args.Topic != "" {
		topic = args.Topic
	}

	msg := args.Message.toProto()

	if err = server.AppendRLNProof(r.node, msg); err != nil {
		return err
	}

	_, err = r.node.Relay().Publish(req.Context(), msg, relay.WithPubSubTopic(topic))
	if err != nil {
		r.log.Error("publishing message", zap.Error(err))
		return err
	}

	*reply = true
	return nil
}

// PostV1AutoSubscription is invoked when the json rpc request uses the post_waku_v2_relay_v1_auto_subscription
// Note that this method takes contentTopics as an argument instead of pubsubtopics and uses autosharding to derive pubsubTopics.
func (r *RelayService) PostV1AutoSubscription(req *http.Request, args *TopicsArgs, reply *SuccessReply) error {

	_, err := r.node.Relay().Subscribe(r.node.Relay().Context(), protocol.NewContentFilter("", args.Topics...))
	if err != nil {
		r.log.Error("subscribing to topics", zap.Strings("topics", args.Topics), zap.Error(err))
		return err
	}
	//TODO: Handle partial errors.

	*reply = true
	return nil
}

// DeleteV1AutoSubscription is invoked when the json rpc request uses the delete_waku_v2_relay_v1_auto_subscription
// Note that this method takes contentTopics as an argument instead of pubsubtopics and uses autosharding to derive pubsubTopics.
func (r *RelayService) DeleteV1AutoSubscription(req *http.Request, args *TopicsArgs, reply *SuccessReply) error {
	ctx := req.Context()

	err := r.node.Relay().Unsubscribe(ctx, protocol.NewContentFilter("", args.Topics...))
	if err != nil {
		r.log.Error("unsubscribing from topics", zap.Strings("topic", args.Topics), zap.Error(err))
		return err
	}
	//TODO: Handle partial errors.
	*reply = true
	return nil
}

// PostV1AutoMessage is invoked when the json rpc request uses the post_waku_v2_relay_v1_auto_message
func (r *RelayService) PostV1AutoMessage(req *http.Request, args *RelayAutoMessageArgs, reply *SuccessReply) error {
	var err error
	msg := args.Message.toProto()
	if msg == nil {
		err := fmt.Errorf("invalid message format received")
		r.log.Error("publishing message", zap.Error(err))
		return err
	}

	if err = server.AppendRLNProof(r.node, msg); err != nil {
		return err
	}

	_, err = r.node.Relay().Publish(req.Context(), msg)
	if err != nil {
		r.log.Error("publishing message", zap.Error(err))
		return err
	}

	*reply = true
	return nil
}

// GetV1AutoMessages is invoked when the json rpc request uses the get_waku_v2_relay_v1_auto_messages method
// Note that this method takes contentTopic as an argument instead of pubSubtopic and uses autosharding.
func (r *RelayService) GetV1AutoMessages(req *http.Request, args *TopicArgs, reply *MessagesReply) error {
	sub, err := r.node.Relay().GetSubscription(args.Topic)
	if err != nil {
		return err
	}
	select {
	case msg := <-sub.Ch:
		*reply = append(*reply, ProtoToRPC(msg.Message()))
	default:
		break
	}
	return nil
}

// PostV1Subscription is invoked when the json rpc request uses the post_waku_v2_relay_v1_subscription method
func (r *RelayService) PostV1Subscription(req *http.Request, args *TopicsArgs, reply *SuccessReply) error {
	ctx := req.Context()

	for _, topic := range args.Topics {
		var err error
		if topic == "" {
			topic = relay.DefaultWakuTopic
		}
		var sub *relay.Subscription
		subs, err := r.node.Relay().Subscribe(ctx, protocol.NewContentFilter(topic))
		if err != nil {
			r.log.Error("subscribing to topic", zap.String("topic", topic), zap.Error(err))
			return err
		}
		sub = subs[0]
		sub.Unsubscribe()
		r.messagesMutex.Lock()
		r.messages[topic] = make([]*pb.WakuMessage, 0)
		r.messagesMutex.Unlock()
	}

	*reply = true
	return nil
}

// DeleteV1Subscription is invoked when the json rpc request uses the delete_waku_v2_relay_v1_subscription method
func (r *RelayService) DeleteV1Subscription(req *http.Request, args *TopicsArgs, reply *SuccessReply) error {
	ctx := req.Context()
	for _, topic := range args.Topics {
		err := r.node.Relay().Unsubscribe(ctx, protocol.NewContentFilter(topic))
		if err != nil {
			r.log.Error("unsubscribing from topic", zap.String("topic", topic), zap.Error(err))
			return err
		}

		delete(r.messages, topic)
	}

	*reply = true
	return nil
}

// GetV1Messages is invoked when the json rpc request uses the get_waku_v2_relay_v1_messages method
func (r *RelayService) GetV1Messages(req *http.Request, args *TopicArgs, reply *MessagesReply) error {
	r.messagesMutex.Lock()
	defer r.messagesMutex.Unlock()

	if _, ok := r.messages[args.Topic]; !ok {
		return fmt.Errorf("topic %s not subscribed", args.Topic)
	}

	for i := range r.messages[args.Topic] {
		*reply = append(*reply, ProtoToRPC(r.messages[args.Topic][i]))
	}

	r.messages[args.Topic] = make([]*pb.WakuMessage, 0)

	return nil
}
