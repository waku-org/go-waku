package rpc

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/status-im/go-waku/waku/v2/node"
	"github.com/status-im/go-waku/waku/v2/protocol"
	"github.com/status-im/go-waku/waku/v2/protocol/pb"
)

type RelayService struct {
	node *node.WakuNode

	messages      map[string][]*pb.WakuMessage
	messagesMutex sync.RWMutex

	ch   chan *protocol.Envelope
	quit chan bool
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

type MessagesReply struct {
	Messages []*pb.WakuMessage `json:"messages,omitempty"`
}

func NewRelayService(node *node.WakuNode) *RelayService {
	return &RelayService{
		node:     node,
		messages: make(map[string][]*pb.WakuMessage),
		quit:     make(chan bool),
	}
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
	r.ch = make(chan *protocol.Envelope, 1024)
	r.node.Broadcaster().Register(r.ch)

	for {
		select {
		case <-r.quit:
			return
		case envelope := <-r.ch:
			r.addEnvelope(envelope)
		}
	}
}

func (r *RelayService) Stop() {
	r.quit <- true
	r.node.Broadcaster().Unregister(r.ch)
	close(r.ch)
}

func (r *RelayService) PostV1Message(req *http.Request, args *RelayMessageArgs, reply *SuccessReply) error {
	_, err := r.node.Relay().Publish(req.Context(), &args.Message, &args.Topic)
	if err != nil {
		log.Error("Error publishing message:", err)
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
		_, err := r.node.Relay().Subscribe(ctx, &topic)
		if err != nil {
			log.Error("Error subscribing to topic:", topic, "err:", err)
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
			log.Error("Error unsubscribing from topic:", topic, "err:", err)
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
