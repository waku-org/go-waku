package main

import (
	"chat2-reliable/pb"
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/bits-and-blooms/bloom/v3"
	"github.com/waku-org/go-waku/waku/v2/peermanager"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	wpb "github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/protocol/store"
)

const (
	bloomFilterSize     = 10000
	bloomFilterFPRate   = 0.01
	bufferSweepInterval = 5 * time.Second
	syncMessageInterval = 30 * time.Second
	messageAckTimeout   = 10 * time.Second
)

func (c *Chat) processReceivedMessage(msg *pb.Message) {
	// Check if the message is already in the bloom filter
	if c.bloomFilter.Test([]byte(msg.MessageId)) {
		// Review ACK status of messages in the unacknowledged outgoing buffer
		c.reviewAckStatus(msg)
		return
	}

	// Update Lamport timestamp
	c.updateLamportTimestamp(msg.LamportTimestamp)

	// Update bloom filter
	c.updateBloomFilter(msg.MessageId)

	// Check causal dependencies
	if c.checkCausalDependencies(msg) {
		if msg.Content != "" {
			// Process the message
			c.ui.ChatMessage(int64(c.getLamportTimestamp()), msg.SenderId, msg.Content)
		}

		// Add to message history
		c.addToMessageHistory(msg)

		// Update received bloom filter for the sender
		if msg.BloomFilter != nil {
			receivedFilter := &bloom.BloomFilter{}
			err := receivedFilter.UnmarshalBinary(msg.BloomFilter)
			if err == nil {
				c.updateReceivedBloomFilter(msg.SenderId, receivedFilter)
			}
		}
	} else {
		// Request missing dependencies
		for _, depID := range msg.CausalHistory {
			if !c.bloomFilter.Test([]byte(depID)) {
				c.requestMissingMessage(depID)
			}
		}
		// Add to incoming buffer
		c.addIncomingBuffer(msg)
	}
}

func (c *Chat) reviewAckStatus(msg *pb.Message) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Review causal history
	for _, msgID := range msg.CausalHistory {
		for i, outMsg := range c.outgoingBuffer {
			if outMsg.MessageId == msgID {
				// acknowledged and remove from outgoing buffer
				c.outgoingBuffer = append(c.outgoingBuffer[:i], c.outgoingBuffer[i+1:]...)
				break
			}
		}
	}

	// Review bloom filter
	if msg.BloomFilter != nil {
		receivedFilter := &bloom.BloomFilter{}
		err := receivedFilter.UnmarshalBinary(msg.BloomFilter)
		if err == nil {
			for i, outMsg := range c.outgoingBuffer {
				if receivedFilter.Test([]byte(outMsg.MessageId)) {
					// possibly acknowledged and remove it from the outgoing buffer
					c.outgoingBuffer = append(c.outgoingBuffer[:i], c.outgoingBuffer[i+1:]...)
				}
			}
		}
	}
}

func (c *Chat) requestMissingMessage(messageID string) {
	// Implement logic to request a missing message from Store nodes or other participants
	//go func() {
	ctx, cancel := context.WithTimeout(c.ctx, 10*time.Second)
	defer cancel()

	hash, err := base64.URLEncoding.DecodeString(messageID)
	if err != nil {
		c.ui.ErrorMessage(fmt.Errorf("failed to parse message hash: %w", err))
		return
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
		c.ui.ErrorMessage(fmt.Errorf("failed to find a store node: %w", err))
		return
	}
	response, err := c.node.Store().Request(ctx, x,
		store.WithAutomaticRequestID(),
		store.WithPeer(peers[0]),
		//store.WithAutomaticPeerSelection(),
		store.WithPaging(true, 100), // Use paging to handle potentially large result sets
	)

	if err != nil {
		c.ui.ErrorMessage(fmt.Errorf("failed to retrieve missing message: %w", err))
		return
	}

	// Filter the response to find the specific message
	for _, msg := range response.Messages() {
		decodedMsg, err := decodeMessage(c.options.ContentTopic, msg.Message)
		if err != nil {
			continue
		}
		if decodedMsg.MessageId == messageID {
			c.C <- protocol.NewEnvelope(msg.Message, msg.Message.GetTimestamp(), relay.DefaultWakuTopic)
			return
		}
	}

	c.ui.ErrorMessage(fmt.Errorf("missing message not found: %s", messageID))
	//}()
}

func (c *Chat) checkCausalDependencies(msg *pb.Message) bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	for _, depID := range msg.CausalHistory {
		if !c.bloomFilter.Test([]byte(depID)) {
			return false
		}
	}
	return true
}

func (c *Chat) updateBloomFilter(messageID string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.bloomFilter.Add([]byte(messageID))
}

func (c *Chat) addToMessageHistory(msg *pb.Message) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.messageHistory = append(c.messageHistory, msg)
	if len(c.messageHistory) > maxMessageHistory {
		c.messageHistory = c.messageHistory[1:]
	}
}

func (c *Chat) periodicBufferSweep() {
	defer c.wg.Done()

	ticker := time.NewTicker(bufferSweepInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.sweepBuffers()
		}
	}
}

func (c *Chat) sweepBuffers() {
	// Process incoming buffer
	newIncomingBuffer := make([]*pb.Message, 0)
	for _, msg := range c.incomingBuffer {
		if c.checkCausalDependencies(msg) {
			c.processReceivedMessage(msg)
		} else {
			newIncomingBuffer = append(newIncomingBuffer, msg)
		}
	}
	c.setIncomingBuffer(newIncomingBuffer)

	// Resend unacknowledged messages from outgoing buffer
	for _, msg := range c.outgoingBuffer {
		if !c.isMessageAcknowledged(msg) {
			c.resendMessage(msg)
		}
	}
}

func (c *Chat) isMessageAcknowledged(msg *pb.Message) bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	ackCount := 0
	totalPeers := len(c.receivedBloomFilters)

	for _, filter := range c.receivedBloomFilters {
		if filter.Test([]byte(msg.MessageId)) {
			ackCount++
		}
	}

	// Consider a message acknowledged if at least 2/3 of peers have it in their bloom filter
	return ackCount >= (2 * totalPeers / 3)
}

func (c *Chat) resendMessage(msg *pb.Message) {
	// Implement logic to resend the message
	//go func() {
	ctx, cancel := context.WithTimeout(c.ctx, messageAckTimeout)
	defer cancel()

	err := c.publish(ctx, msg)
	if err != nil {
		c.ui.ErrorMessage(fmt.Errorf("failed to resend message: %w", err))
	}
	//}()
}

func (c *Chat) periodicSyncMessage() {
	defer c.wg.Done()

	ticker := time.NewTicker(syncMessageInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.sendSyncMessage()
		}
	}
}

func (c *Chat) sendSyncMessage() {
	bloomBytes, err := c.bloomFilterBytes()
	if err != nil {
		c.ui.ErrorMessage(fmt.Errorf("failed to marshal bloom filter: %w", err))
		return
	}

	syncMsg := &pb.Message{
		SenderId:         c.node.Host().ID().String(),
		MessageId:        generateUniqueID(),
		LamportTimestamp: c.lamportTimestamp,
		CausalHistory:    c.getRecentMessageIDs(2),
		ChannelId:        c.options.ContentTopic,
		BloomFilter:      bloomBytes,
		Content:          "", // Empty content for sync messages
	}

	err = c.publish(c.ctx, syncMsg)
	if err != nil {
		c.ui.ErrorMessage(fmt.Errorf("failed to send sync message: %w", err))
	}
}

func (c *Chat) addIncomingBuffer(msg *pb.Message) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.incomingBuffer = append(c.incomingBuffer, msg)
}

func (c *Chat) setIncomingBuffer(newBuffer []*pb.Message) {
	c.incomingBuffer = newBuffer
}

func (c *Chat) bloomFilterBytes() ([]byte, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.bloomFilter.MarshalBinary()
}

func (c *Chat) updateReceivedBloomFilter(senderId string, receivedFilter *bloom.BloomFilter) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.receivedBloomFilters[senderId] = receivedFilter
}

func (c *Chat) incLamportTimestamp() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.lamportTimestamp++
}

func (c *Chat) updateLamportTimestamp(msgTs int32) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if msgTs > c.lamportTimestamp {
		c.lamportTimestamp = msgTs
	}
}

func (c *Chat) getLamportTimestamp() int32 {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.lamportTimestamp
}
