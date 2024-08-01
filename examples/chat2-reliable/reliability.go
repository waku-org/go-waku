package main

import (
	"chat2-reliable/pb"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"sort"
	"sync"
	"time"

	"github.com/bits-and-blooms/bloom/v3"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/waku-org/go-waku/waku/v2/peermanager"
	wpb "github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/protocol/store"
	"google.golang.org/protobuf/proto"
)

const (
	bloomFilterSize          = 10000
	bloomFilterFPRate        = 0.01
	bloomFilterWindow        = 1 * time.Hour
	bloomFilterCleanInterval = 5 * time.Minute
	bufferSweepInterval      = 5 * time.Second
	syncMessageInterval      = 30 * time.Second
	messageAckTimeout        = 10 * time.Second
	maxRetries               = 3
	retryBaseDelay           = 1 * time.Second
	maxRetryDelay            = 10 * time.Second
	ackTimeout               = 5 * time.Second
	maxResendAttempts        = 5
	resendBaseDelay          = 1 * time.Second
	maxResendDelay           = 30 * time.Second
)

type TimestampedMessageID struct {
	ID        string
	Timestamp time.Time
}

type RollingBloomFilter struct {
	filter   *bloom.BloomFilter
	window   time.Duration
	messages []TimestampedMessageID
	mutex    sync.Mutex
}

type UnacknowledgedMessage struct {
	Message        *pb.Message
	SendTime       time.Time
	ResendAttempts int
}

func (c *Chat) initReliabilityProtocol() {
	c.wg.Add(4)
	go c.startBloomFilterCleaner()
	go c.periodicBufferSweep()
	go c.periodicSyncMessage()
	go c.startEagerPushMechanism()
}

func (c *Chat) startEagerPushMechanism() {
	defer c.wg.Done()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.checkUnacknowledgedMessages()
		}
	}
}

func NewRollingBloomFilter() *RollingBloomFilter {
	return &RollingBloomFilter{
		filter:   bloom.NewWithEstimates(bloomFilterSize, bloomFilterFPRate),
		window:   bloomFilterWindow,
		messages: make([]TimestampedMessageID, 0),
	}
}

func (rbf *RollingBloomFilter) Add(messageID string) {
	rbf.mutex.Lock()
	defer rbf.mutex.Unlock()

	rbf.filter.Add([]byte(messageID))
	rbf.messages = append(rbf.messages, TimestampedMessageID{
		ID:        messageID,
		Timestamp: time.Now(),
	})
}

func (rbf *RollingBloomFilter) Test(messageID string) bool {
	rbf.mutex.Lock()
	defer rbf.mutex.Unlock()

	return rbf.filter.Test([]byte(messageID))
}

func (rbf *RollingBloomFilter) Clean() {
	rbf.mutex.Lock()
	defer rbf.mutex.Unlock()

	cutoff := time.Now().Add(-rbf.window)
	newMessages := make([]TimestampedMessageID, 0)
	newFilter := bloom.NewWithEstimates(bloomFilterSize, bloomFilterFPRate)

	for _, msg := range rbf.messages {
		if msg.Timestamp.After(cutoff) {
			newMessages = append(newMessages, msg)
			newFilter.Add([]byte(msg.ID))
		}
	}

	rbf.messages = newMessages
	rbf.filter = newFilter
}

func (c *Chat) startBloomFilterCleaner() {
	defer c.wg.Done()

	ticker := time.NewTicker(bloomFilterCleanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.bloomFilter.Clean()
		}
	}
}

func (c *Chat) processReceivedMessage(msg *pb.Message) {
	// Check if the message is already in the bloom filter
	if c.bloomFilter.Test(msg.MessageId) {
		// Review ACK status of messages in the unacknowledged outgoing buffer
		c.reviewAckStatus(msg)
		return
	}

	// Update Lamport timestamp
	c.updateLamportTimestamp(msg.LamportTimestamp)

	// Update bloom filter
	c.bloomFilter.Add(msg.MessageId)

	// Check causal dependencies
	missingDeps := c.checkCausalDependencies(msg)
	if len(missingDeps) == 0 {
		if msg.Content != "" {
			// Process the message
			c.ui.ChatMessage(int64(c.getLamportTimestamp()), msg.SenderId, msg.Content)
		}

		// Add to message history
		c.addToMessageHistory(msg)

		// Process any messages in the buffer that now have their dependencies met
		c.processBufferedMessages()
	} else {
		// Request missing dependencies
		for _, depID := range missingDeps {
			c.requestMissingMessage(depID)
		}
		// Add to incoming buffer
		c.addToIncomingBuffer(msg)
	}
}

func (c *Chat) processBufferedMessages() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	processed := make(map[string]bool)
	for {
		madeProgress := false
		for i := 0; i < len(c.incomingBuffer); i++ {
			msg := c.incomingBuffer[i]
			if processed[msg.MessageId] {
				continue
			}
			if c.checkCausalDependencies(msg) == nil {
				c.processReceivedMessage(msg)
				processed[msg.MessageId] = true
				c.incomingBuffer = append(c.incomingBuffer[:i], c.incomingBuffer[i+1:]...)
				i--
				madeProgress = true
			}
		}
		if !madeProgress {
			break
		}
	}
}

func (c *Chat) reviewAckStatus(msg *pb.Message) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Review causal history
	for _, msgID := range msg.CausalHistory {
		for i, outMsg := range c.outgoingBuffer {
			if outMsg.Message.MessageId == msgID {
				// acknowledged and remove from outgoing buffer
				c.outgoingBuffer = append(c.outgoingBuffer[:i], c.outgoingBuffer[i+1:]...)
				break
			}
		}
	}

	// Review bloom filter
	if msg.BloomFilter != nil {
		receivedFilter := bloom.NewWithEstimates(bloomFilterSize, bloomFilterFPRate)
		err := receivedFilter.UnmarshalBinary(msg.BloomFilter)
		if err == nil {
			for i := 0; i < len(c.outgoingBuffer); i++ {
				if receivedFilter.Test([]byte(c.outgoingBuffer[i].Message.MessageId)) {
					// possibly acknowledged and remove it from the outgoing buffer
					c.outgoingBuffer = append(c.outgoingBuffer[:i], c.outgoingBuffer[i+1:]...)
					i--
				}
			}
		}
	}
}

func (c *Chat) requestMissingMessage(messageID string) {
	for retry := 0; retry < maxRetries; retry++ {
		err := c.doRequestMissingMessageFromPeers(messageID)
		if err == nil {
			return
		}

		c.ui.ErrorMessage(fmt.Errorf("failed to retrieve missing message (attempt %d): %w", retry+1, err))

		// Exponential backoff
		delay := retryBaseDelay * time.Duration(1<<uint(retry))
		if delay > maxRetryDelay {
			delay = maxRetryDelay
		}
		time.Sleep(delay)
	}

	c.ui.ErrorMessage(fmt.Errorf("failed to retrieve missing message after %d attempts: %s", maxRetries, messageID))
}

func (c *Chat) doRequestMissingMessage(messageID string) error {
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
	// alternate below
	// criteria := store.MessageHashCriteria{
	// 	MessageHashes: []wpb.MessageHash{wpb.ToMessageHash(hash)},
	// }

	// response, err := c.node.Store().Request(ctx, criteria,
	// 	store.WithAutomaticRequestID(),
	// 	store.WithAutomaticPeerSelection(),
	// 	store.WithPaging(true, 100),
	// )

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

func (c *Chat) checkCausalDependencies(msg *pb.Message) []string {
	var missingDeps []string
	for _, depID := range msg.CausalHistory {
		if !c.bloomFilter.Test(depID) {
			missingDeps = append(missingDeps, depID)
		}
	}
	return missingDeps
}

func (c *Chat) addToMessageHistory(msg *pb.Message) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.messageHistory = append(c.messageHistory, msg)
	c.messageHistory = c.resolveConflicts(c.messageHistory)

	if len(c.messageHistory) > maxMessageHistory {
		c.messageHistory = c.messageHistory[len(c.messageHistory)-maxMessageHistory:]
	}
}

func (c *Chat) resolveConflicts(messages []*pb.Message) []*pb.Message {
	// Group messages by Lamport timestamp
	groupedMessages := make(map[int32][]*pb.Message)
	for _, msg := range messages {
		groupedMessages[msg.LamportTimestamp] = append(groupedMessages[msg.LamportTimestamp], msg)
	}

	// Sort timestamps
	var timestamps []int32
	for ts := range groupedMessages {
		timestamps = append(timestamps, ts)
	}
	sort.Slice(timestamps, func(i, j int) bool { return timestamps[i] < timestamps[j] })

	// Resolve conflicts and create a new ordered list
	var resolvedMessages []*pb.Message
	for _, ts := range timestamps {
		msgs := groupedMessages[ts]
		if len(msgs) == 1 {
			resolvedMessages = append(resolvedMessages, msgs[0])
		} else {
			// Sort conflicting messages by MessageId
			sort.Slice(msgs, func(i, j int) bool { return msgs[i].MessageId < msgs[j].MessageId })
			resolvedMessages = append(resolvedMessages, msgs...)
		}
	}

	return resolvedMessages
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
	c.mutex.Lock()
	newIncomingBuffer := make([]*pb.Message, 0)
	for _, msg := range c.incomingBuffer {
		missingDeps := c.checkCausalDependencies(msg)
		if len(missingDeps) == 0 {
			c.processReceivedMessage(msg)
		} else {
			newIncomingBuffer = append(newIncomingBuffer, msg)
		}
	}
	c.incomingBuffer = newIncomingBuffer
	c.mutex.Unlock()

	// Resend unacknowledged messages from outgoing buffer
	c.checkUnacknowledgedMessages()
}

func (c *Chat) checkUnacknowledgedMessages() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()
	for i := 0; i < len(c.outgoingBuffer); i++ {
		unackMsg := c.outgoingBuffer[i]
		if now.Sub(unackMsg.SendTime) > ackTimeout {
			if unackMsg.ResendAttempts < maxResendAttempts {
				c.resendMessage(unackMsg.Message)
				c.outgoingBuffer[i].ResendAttempts++
				c.outgoingBuffer[i].SendTime = now
			} else {
				// Remove the message from the buffer after max attempts
				c.outgoingBuffer = append(c.outgoingBuffer[:i], c.outgoingBuffer[i+1:]...)
				i-- // Adjust index after removal
				c.ui.ErrorMessage(fmt.Errorf("message %s failed to be acknowledged after %d attempts", unackMsg.Message.MessageId, maxResendAttempts))
			}
		}
	}
}

func (c *Chat) resendMessage(msg *pb.Message) {
	go func() {
		delay := resendBaseDelay * time.Duration(1<<uint(c.getResendAttempts(msg.MessageId)))
		if delay > maxResendDelay {
			delay = maxResendDelay
		}
		time.Sleep(delay)

		ctx, cancel := context.WithTimeout(c.ctx, ackTimeout)
		defer cancel()

		err := c.publish(ctx, msg)
		if err != nil {
			c.ui.ErrorMessage(fmt.Errorf("failed to resend message: %w", err))
		}
	}()
}

func (c *Chat) getResendAttempts(messageId string) int {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	for _, unackMsg := range c.outgoingBuffer {
		if unackMsg.Message.MessageId == messageId {
			return unackMsg.ResendAttempts
		}
	}
	return 0
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
	bloomBytes, err := c.bloomFilter.MarshalBinary()
	if err != nil {
		c.ui.ErrorMessage(fmt.Errorf("failed to marshal bloom filter: %w", err))
		return
	}

	syncMsg := &pb.Message{
		SenderId:         c.node.Host().ID().String(),
		MessageId:        generateUniqueID(),
		LamportTimestamp: c.getLamportTimestamp(),
		CausalHistory:    c.getRecentMessageIDs(10),
		ChannelId:        c.options.ContentTopic,
		BloomFilter:      bloomBytes,
		Content:          "", // Empty content for sync messages
	}

	ctx, cancel := context.WithTimeout(c.ctx, messageAckTimeout)
	defer cancel()

	err = c.publish(ctx, syncMsg)
	if err != nil {
		c.ui.ErrorMessage(fmt.Errorf("failed to send sync message: %w", err))
	}
}

func (c *Chat) addToIncomingBuffer(msg *pb.Message) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.incomingBuffer = append(c.incomingBuffer, msg)
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
		c.lamportTimestamp = msgTs + 1
	} else {
		c.lamportTimestamp++
	}
}

func (c *Chat) getLamportTimestamp() int32 {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.lamportTimestamp
}

// MarshalBinary implements the encoding.BinaryMarshaler interface for RollingBloomFilter
func (rbf *RollingBloomFilter) MarshalBinary() ([]byte, error) {
	rbf.mutex.Lock()
	defer rbf.mutex.Unlock()
	return rbf.filter.MarshalBinary()
}

// UnmarshalBinary implements the encoding.BinaryUnmarshaler interface for RollingBloomFilter
func (rbf *RollingBloomFilter) UnmarshalBinary(data []byte) error {
	rbf.mutex.Lock()
	defer rbf.mutex.Unlock()
	return rbf.filter.UnmarshalBinary(data)
}

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
	ctx, cancel := context.WithTimeout(c.ctx, 10*time.Second)
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
	return err
}

func readProtobufMessage(stream network.Stream, msg proto.Message) error {
	data, err := io.ReadAll(stream)
	if err != nil {
		return err
	}
	return proto.Unmarshal(data, msg)
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
	}
}

func (c *Chat) setupMessageRequestHandler() {
	c.node.Host().SetStreamHandler(protocol.ID("/chat2-reliable/message-request/1.0.0"), c.handleMessageRequest)
}
