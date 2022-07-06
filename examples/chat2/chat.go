package main

import (
	"chat2/pb"
	"context"
	"crypto/sha256"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/status-im/go-waku/waku/v2/node"
	"github.com/status-im/go-waku/waku/v2/protocol"
	"github.com/status-im/go-waku/waku/v2/protocol/filter"
	wpb "github.com/status-im/go-waku/waku/v2/protocol/pb"
	"github.com/status-im/go-waku/waku/v2/protocol/relay"
	"github.com/status-im/go-waku/waku/v2/utils"
	"golang.org/x/crypto/pbkdf2"
)

// Chat represents a subscription to a single PubSub topic. Messages
// can be published to the topic with Chat.Publish, and received
// messages are pushed to the Messages channel.
type Chat struct {
	// Messages is a channel of messages received from other peers in the chat room
	Messages chan *pb.Chat2Message

	C    chan *protocol.Envelope
	node *node.WakuNode

	self         peer.ID
	contentTopic string
	useV1Payload bool
	useLightPush bool
	nick         string
	spamChan     chan *wpb.WakuMessage
}

// NewChat tries to subscribe to the PubSub topic for the room name, returning
// a ChatRoom on success.
func NewChat(ctx context.Context, n *node.WakuNode, selfID peer.ID, contentTopic string, useV1Payload bool, useLightPush bool, nickname string, spamChan chan *wpb.WakuMessage) (*Chat, error) {
	// join the default waku topic and subscribe to it

	chat := &Chat{
		node:         n,
		self:         selfID,
		contentTopic: contentTopic,
		nick:         nickname,
		useV1Payload: useV1Payload,
		useLightPush: useLightPush,
		Messages:     make(chan *pb.Chat2Message, 1024),
		spamChan:     spamChan,
	}

	if useLightPush {
		cf := filter.ContentFilter{
			Topic:         relay.DefaultWakuTopic,
			ContentTopics: []string{contentTopic},
		}
		var err error
		_, theFilter, err := n.Filter().Subscribe(ctx, cf)
		chat.C = theFilter.Chan
		if err != nil {
			return nil, err
		}
	} else {
		sub, err := n.Relay().Subscribe(ctx)
		if err != nil {
			return nil, err
		}
		chat.C = sub.C
	}

	// start reading messages from the subscription in a loop
	go chat.readLoop()

	go chat.readSpamMessages()

	return chat, nil
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
	var t = time.Now()
	var timestamp int64 = utils.GetUnixEpochFrom(t)
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

	if cr.node.RLNRelay() != nil {
		// for future version when we support more than one rln protected content topic,
		// we should check the message content topic as well
		err = cr.node.RLNRelay().AppendRLNProof(wakuMsg, t)
		if err != nil {
			return err
		}
	}

	if cr.useLightPush {
		_, err = cr.node.Lightpush().Publish(ctx, wakuMsg)

	} else {
		_, err = cr.node.Relay().Publish(ctx, wakuMsg)

	}

	return err
}

func DecodeMessage(useV1Payload bool, contentTopic string, wakumsg *wpb.WakuMessage) (*pb.Chat2Message, error) {
	var keyInfo *node.KeyInfo = &node.KeyInfo{}
	if useV1Payload { // Use WakuV1 encryption
		keyInfo.Kind = node.Symmetric
		keyInfo.SymKey = generateSymKey(contentTopic)
	} else {
		keyInfo.Kind = node.None
	}

	payload, err := node.DecodePayload(wakumsg, keyInfo)
	if err != nil {
		return nil, err
	}

	msg := &pb.Chat2Message{}
	if err := proto.Unmarshal(payload.Data, msg); err != nil {
		return nil, err
	}

	return msg, nil
}

// readLoop pulls messages from the pubsub topic and pushes them onto the Messages channel.
func (cr *Chat) readLoop() {
	for value := range cr.C {
		msg, err := DecodeMessage(cr.useV1Payload, cr.contentTopic, value.Message())
		if err == nil {
			// send valid messages onto the Messages channel
			cr.Messages <- msg
		}
	}
}

// readSpam prints messages that are spam (to demonstrate RLN functionality)
func (cr *Chat) readSpamMessages() {
	for value := range cr.C {
		msg, err := DecodeMessage(cr.useV1Payload, cr.contentTopic, value.Message())
		if err == nil {
			msg.Payload = append([]byte("Spam message received and discarded: "), msg.Payload...)
			cr.Messages <- msg
		}
	}
}

func (cr *Chat) displayMessages(messages []*wpb.WakuMessage) {
	for _, msg := range messages {
		msg, err := DecodeMessage(cr.useV1Payload, cr.contentTopic, msg)
		if err == nil {
			// send valid messages onto the Messages channel
			cr.Messages <- msg
		}
	}
}
