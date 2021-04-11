package main

import (
	"chat2/pb"
	"context"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/status-im/go-waku/waku/v2/node"
	"github.com/status-im/go-waku/waku/v2/protocol"
)

// Chat represents a subscription to a single PubSub topic. Messages
// can be published to the topic with Chat.Publish, and received
// messages are pushed to the Messages channel.
type Chat struct {
	// Messages is a channel of messages received from other peers in the chat room
	Messages chan *pb.Chat2Message

	ctx  context.Context
	sub  *node.Subscription
	node *node.WakuNode

	self peer.ID
	nick string
}

// NewChat tries to subscribe to the PubSub topic for the room name, returning
// a ChatRoom on success.
func NewChat(ctx context.Context, n *node.WakuNode, selfID peer.ID, nickname string) (*Chat, error) {
	// join the default waku topic and subscribe to it
	sub, err := n.Subscribe(nil)
	if err != nil {
		return nil, err
	}

	c := &Chat{
		ctx:      ctx,
		node:     n,
		sub:      sub,
		self:     selfID,
		nick:     nickname,
		Messages: make(chan *pb.Chat2Message, 1024),
	}

	// start reading messages from the subscription in a loop
	go c.readLoop()

	return c, nil
}

// Publish sends a message to the pubsub topic.
func (cr *Chat) Publish(message string) error {

	msg := &pb.Chat2Message{
		Timestamp: uint64(time.Now().Unix()),
		Nick:      cr.nick,
		Payload:   []byte(message),
	}

	msgBytes, err := proto.Marshal(msg)
	if err != nil {
		return err
	}

	var version uint32 = 0
	var timestamp float64 = float64(time.Now().UnixNano())
	var keyInfo *node.KeyInfo = &node.KeyInfo{Kind: node.None}

	p := new(node.Payload)
	p.Data = msgBytes
	p.Key = keyInfo

	payload, err := p.Encode(0)
	if err != nil {
		return err
	}

	wakuMsg := &protocol.WakuMessage{
		Payload:      payload,
		Version:      version,
		ContentTopic: DefaultContentTopic,
		Timestamp:    timestamp,
	}

	_, err = cr.node.Publish(wakuMsg, nil)

	return err
}

// readLoop pulls messages from the pubsub topic and pushes them onto the Messages channel.
func (cr *Chat) readLoop() {
	for value := range cr.sub.C {
		payload, err := node.DecodePayload(value.Message(), &node.KeyInfo{Kind: node.None})
		if err != nil {
			continue
		}

		msg := &pb.Chat2Message{}
		if err := proto.Unmarshal(payload.Data, msg); err != nil {
			continue
		}

		// send valid messages onto the Messages channel
		cr.Messages <- msg
	}
}
