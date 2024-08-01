package main

import (
	"chat2-reliable/pb"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/waku-org/go-waku/waku/v2/peermanager"
	wpb "github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/protocol/store"
	"google.golang.org/protobuf/proto"
)

// below functions are specifically for peer retrieval of missing msgs instead of store
func (c *Chat) doRequestMissingMessageFromPeers(messageID string) error {
	peers := c.node.Host().Network().Peers()
	for _, peerID := range peers {
		msg, err := c.requestMessageFromPeer(peerID, messageID)
		if err == nil && msg != nil {
			c.processReceivedMessage(msg)
			return nil
		}
	}
	return fmt.Errorf("no peers could provide the missing message")
}

func (c *Chat) requestMessageFromPeer(peerID peer.ID, messageID string) (*pb.Message, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	stream, err := c.node.Host().NewStream(ctx, peerID, protocol.ID("/chat2-reliable/message-request/1.0.0"))
	if err != nil {
		return nil, fmt.Errorf("failed to open stream to peer: %w", err)
	}
	defer stream.Close()

	// Send message request
	request := &pb.MessageRequest{MessageId: messageID}
	err = writeProtobufMessage(stream, request)
	if err != nil {
		return nil, fmt.Errorf("failed to send message request: %w", err)
	}

	// Read response
	response := &pb.MessageResponse{}
	err = readProtobufMessage(stream, response)
	if err != nil {
		return nil, fmt.Errorf("failed to read message response: %w", err)
	}

	fmt.Printf("Received message response from peer %s\n", peerID.String())

	if response.Message == nil {
		return nil, fmt.Errorf("peer did not have the requested message")
	}
	return response.Message, nil
}

// Helper functions for protobuf message reading/writing
func writeProtobufMessage(stream network.Stream, msg proto.Message) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		return err
	}

	_, err = stream.Write(data)
	if err != nil {
		return err
	}

	// Add a delay before closing the stream
	time.Sleep(1 * time.Second)

	err = stream.Close()
	if err != nil {
		return err
	}
	return nil
}

func readProtobufMessage(stream network.Stream, msg proto.Message) error {
	err := stream.SetReadDeadline(time.Now().Add(30 * time.Second))
	if err != nil {
		return fmt.Errorf("failed to set read deadline: %w", err)
	}

	var data []byte
	buffer := make([]byte, 1024)
	for {
		n, err := stream.Read(buffer)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		data = append(data, buffer[:n]...)
	}

	err = proto.Unmarshal(data, msg)
	if err != nil {
		return err
	}
	return nil
}

func (c *Chat) handleMessageRequest(stream network.Stream) {
	defer stream.Close()

	request := &pb.MessageRequest{}
	err := readProtobufMessage(stream, request)
	if err != nil {
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
	err = writeProtobufMessage(stream, response)
	if err != nil {
		c.ui.ErrorMessage(fmt.Errorf("failed to send message response: %w", err))
		return
	}
}

func (c *Chat) setupMessageRequestHandler() {
	c.node.Host().SetStreamHandler(protocol.ID("/chat2-reliable/message-request/1.0.0"), c.handleMessageRequest)
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
