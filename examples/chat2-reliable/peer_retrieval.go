package main

import (
	"chat2-reliable/pb"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-msgio/pbio"
	"github.com/waku-org/go-waku/waku/v2/peermanager"
	wpb "github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/protocol/store"
	"google.golang.org/protobuf/proto"
)

const messageRequestProtocolID = protocol.ID("/chat2-reliable/message-request/1.0.0")

// below functions are specifically for peer retrieval of missing msgs instead of store
func (c *Chat) doRequestMissingMessageFromPeers(messageID string) (*pb.Message, error) {
	peers := c.node.Host().Network().Peers()
	for _, peerID := range peers {
		msg, err := c.requestMessageFromPeer(peerID, messageID)
		if err == nil && msg != nil {
			return msg, nil
		}
	}
	return nil, errors.New("no peers could provide the missing message")
}

func (c *Chat) requestMessageFromPeer(peerID peer.ID, messageID string) (*pb.Message, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	stream, err := c.node.Host().NewStream(ctx, peerID, messageRequestProtocolID)
	if err != nil {
		return nil, fmt.Errorf("failed to open stream to peer: %w", err)
	}

	writer := pbio.NewDelimitedWriter(stream)
	reader := pbio.NewDelimitedReader(stream, math.MaxInt32)

	// Send message request
	request := &pb.MessageRequest{MessageId: messageID}
	err = writeProtobufMessage(writer, request)
	if err != nil {
		return nil, fmt.Errorf("failed to send message request: %w", err)
	}

	// Read response
	response := &pb.MessageResponse{}
	err = readProtobufMessage(reader, response)
	if err != nil {
		return nil, fmt.Errorf("failed to read message response: %w", err)
	}

	if response.Message == nil {
		return nil, fmt.Errorf("peer did not have the requested message")
	}

	return response.Message, nil
}

// Helper functions for protobuf message reading/writing
func writeProtobufMessage(stream pbio.WriteCloser, msg proto.Message) error {
	err := stream.WriteMsg(msg)
	if err != nil {
		return err
	}

	return nil
}

func readProtobufMessage(stream pbio.ReadCloser, msg proto.Message) error {
	err := stream.ReadMsg(msg)
	if err != nil {
		return err
	}

	return nil
}

func (c *Chat) handleMessageRequest(stream network.Stream) {
	writer := pbio.NewDelimitedWriter(stream)
	reader := pbio.NewDelimitedReader(stream, math.MaxInt32)

	request := &pb.MessageRequest{}
	err := readProtobufMessage(reader, request)
	if err != nil {
		stream.Reset()
		c.ui.ErrorMessage(fmt.Errorf("failed to read message request: %w", err))
		return
	}

	c.mutex.Lock()
	var foundMessage *pb.Message
	for _, msg := range c.messageHistory {
		if msg.MessageId == request.MessageId {
			foundMessage = msg
			break
		}
	}
	c.mutex.Unlock()

	response := &pb.MessageResponse{Message: foundMessage}
	err = writeProtobufMessage(writer, response)
	if err != nil {
		stream.Reset()
		c.ui.ErrorMessage(fmt.Errorf("failed to send message response: %w", err))
		return
	}

	stream.Close()
}

func (c *Chat) setupMessageRequestHandler() {
	c.node.Host().SetStreamHandler(messageRequestProtocolID, c.handleMessageRequest)
}

func (c *Chat) _doRequestMissingMessageFromStore(messageID string) error {
	ctx, cancel := context.WithTimeout(c.ctx, 10*time.Second)
	defer cancel()

	hash, err := base64.URLEncoding.DecodeString(messageID)
	if err != nil {
		return fmt.Errorf("failed to parse message hash: %w", err)
	}

	x := store.MessageHashCriteria{
		MessageHashes: []wpb.MessageHash{wpb.ToMessageHash(hash)},
	}

	peers, err := c.node.PeerManager().SelectPeers(peermanager.PeerSelectionCriteria{
		SelectionType: peermanager.Automatic,
		Proto:         store.StoreQueryID_v300,
		PubsubTopics:  []string{relay.DefaultWakuTopic},
		Ctx:           ctx,
	})
	if err != nil {
		return fmt.Errorf("failed to find a store node: %w", err)
	}
	response, err := c.node.Store().Request(ctx, x,
		store.WithAutomaticRequestID(),
		store.WithPeer(peers[0]),
		//store.WithAutomaticPeerSelection(),
		store.WithPaging(true, 100), // Use paging to handle potentially large result sets
	)

	if err != nil {
		return fmt.Errorf("failed to retrieve missing message: %w", err)
	}

	for _, msg := range response.Messages() {
		decodedMsg, err := decodeMessage(c.options.ContentTopic, msg.Message)
		if err != nil {
			continue
		}
		if decodedMsg.MessageId == messageID {
			c.processReceivedMessage(decodedMsg)
			return nil
		}
	}

	return fmt.Errorf("missing message not found: %s", messageID)
}
