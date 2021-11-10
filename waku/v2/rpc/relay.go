package rpc

import (
	"net/http"

	"github.com/status-im/go-waku/waku/v2/node"
	"github.com/status-im/go-waku/waku/v2/protocol/pb"
	"github.com/status-im/go-waku/waku/v2/protocol/relay"
)

type RelayService struct {
	node *node.WakuNode
}

type RelayMessageArgs struct {
	Topic   string         `json:"topic,omitempty"`
	Message pb.WakuMessage `json:"message,omitempty"`
}

type TopicsArgs struct {
	Topics []string `json:"topics,omitempty"`
}

func (r *RelayService) PostV1Message(req *http.Request, args *RelayMessageArgs, reply *SuccessReply) error {
	_, err := r.node.Relay().Publish(req.Context(), &args.Message, (*relay.Topic)(&args.Topic))
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
		_, err := r.node.Relay().Subscribe(ctx, (*relay.Topic)(&topic))
		if err != nil {
			log.Error("Error subscribing to topic:", topic, "err:", err)
			reply.Success = false
			reply.Error = err.Error()
			return nil
		}
	}
	reply.Success = true
	return nil
}

func (r *RelayService) DeleteV1Subscription(req *http.Request, args *TopicsArgs, reply *SuccessReply) error {
	ctx := req.Context()
	for _, topic := range args.Topics {
		err := r.node.Relay().Unsubscribe(ctx, (relay.Topic)(topic))
		if err != nil {
			log.Error("Error unsubscribing from topic:", topic, "err:", err)
			reply.Success = false
			reply.Error = err.Error()
			return nil
		}
	}
	reply.Success = true
	return nil
}
