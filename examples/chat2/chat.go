package main

import (
	"chat2/pb"
	"context"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/status-im/go-waku/waku/v2/node"
	wpb "github.com/status-im/go-waku/waku/v2/protocol/pb"
)

// Chat represents a subscription to a single PubSub topic. Messages
// can be published to the topic with Chat.Publish, and received
// messages are pushed to the Messages channel.
type Chat struct {
	// Messages is a channel of messages received from other peers in the chat room
	Messages chan *pb.Chat2Message

	sub  *node.Subscription
	node *node.WakuNode

	self peer.ID
	nick string
}

// NewChat tries to subscribe to the PubSub topic for the room name, returning
// a ChatRoom on success.
func NewChat(n *node.WakuNode, selfID peer.ID, nickname string) (*Chat, error) {
	// join the default waku topic and subscribe to it
	sub, err := n.Subscribe(nil)
	if err != nil {
		return nil, err
	}

	c := &Chat{
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
func (cr *Chat) Publish(ctx context.Context, message string) error {

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

	wakuMsg := &wpb.WakuMessage{
		Payload:      payload,
		Version:      version,
		ContentTopic: DefaultContentTopic,
		Timestamp:    timestamp,
	}

	_, err = cr.node.Publish(ctx, wakuMsg, nil)

	return err
}

func (cr *Chat) decodeMessage(wakumsg *wpb.WakuMessage) {
	payload, err := node.DecodePayload(wakumsg, &node.KeyInfo{Kind: node.None})
	if err != nil {
		return
	}

	msg := &pb.Chat2Message{}
	if err := proto.Unmarshal(payload.Data, msg); err != nil {
		return
	}

	// send valid messages onto the Messages channel
	cr.Messages <- msg
}

// readLoop pulls messages from the pubsub topic and pushes them onto the Messages channel.
func (cr *Chat) readLoop() {
	for value := range cr.sub.C {
		cr.decodeMessage(value.Message())
	}
}

func (cr *Chat) displayMessages(messages []*wpb.WakuMessage) {
	for _, msg := range messages {
		cr.decodeMessage(msg)
	}
}
