package main

import (
	"chat2/pb"
	"context"
	"crypto/sha256"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/status-im/go-waku/waku/v2/node"
	wpb "github.com/status-im/go-waku/waku/v2/protocol/pb"
	"golang.org/x/crypto/pbkdf2"
)

// Chat represents a subscription to a single PubSub topic. Messages
// can be published to the topic with Chat.Publish, and received
// messages are pushed to the Messages channel.
type Chat struct {
	// Messages is a channel of messages received from other peers in the chat room
	Messages chan *pb.Chat2Message

	sub  *node.Subscription
	node *node.WakuNode

	self         peer.ID
	contentTopic string
	useV1Payload bool
	nick         string
}

// NewChat tries to subscribe to the PubSub topic for the room name, returning
// a ChatRoom on success.
func NewChat(n *node.WakuNode, selfID peer.ID, contentTopic string, useV1Payload bool, nickname string) (*Chat, error) {
	// join the default waku topic and subscribe to it
	sub, err := n.Subscribe(nil)
	if err != nil {
		return nil, err
	}

	c := &Chat{
		node:         n,
		sub:          sub,
		self:         selfID,
		contentTopic: contentTopic,
		nick:         nickname,
		useV1Payload: useV1Payload,
		Messages:     make(chan *pb.Chat2Message, 1024),
	}

	// start reading messages from the subscription in a loop
	go c.readLoop()

	return c, nil
}

func generateSymKey(password string) []byte {
	// AesKeyLength represents the length (in bytes) of an private key
	AESKeyLength := 256 / 8
	return pbkdf2.Key([]byte(password), nil, 65356, AESKeyLength, sha256.New)
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

	var version uint32
	var timestamp float64 = float64(time.Now().UnixNano())
	var keyInfo *node.KeyInfo = &node.KeyInfo{}

	if cr.useV1Payload { // Use WakuV1 encryption
		keyInfo.Kind = node.Symmetric
		keyInfo.SymKey = generateSymKey(cr.contentTopic)
		version = 1
	} else {
		keyInfo.Kind = node.None
		version = 0
	}

	p := new(node.Payload)
	p.Data = msgBytes
	p.Key = keyInfo

	payload, err := p.Encode(version)
	if err != nil {
		return err
	}

	wakuMsg := &wpb.WakuMessage{
		Payload:      payload,
		Version:      version,
		ContentTopic: cr.contentTopic,
		Timestamp:    timestamp,
	}

	_, err = cr.node.Publish(ctx, wakuMsg, nil)

	return err
}

func (cr *Chat) decodeMessage(wakumsg *wpb.WakuMessage) {
	var keyInfo *node.KeyInfo = &node.KeyInfo{}
	if cr.useV1Payload { // Use WakuV1 encryption
		keyInfo.Kind = node.Symmetric
		keyInfo.SymKey = generateSymKey(cr.contentTopic)
	} else {
		keyInfo.Kind = node.None
	}

	payload, err := node.DecodePayload(wakumsg, keyInfo)
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
