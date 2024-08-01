package main

import (
	"chat2-reliable/pb"
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/bits-and-blooms/bloom/v3"
)

const (
	bloomFilterSize          = 10000
	bloomFilterFPRate        = 0.01
	bloomFilterWindow        = 1 * time.Hour
	bloomFilterCleanInterval = 5 * time.Minute
	bufferSweepInterval      = 5 * time.Second
	syncMessageInterval      = 60 * time.Second
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
			// Add to message history
			c.addToMessageHistory(msg)
		}

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
	remainingBuffer := make([]*pb.Message, 0, len(c.incomingBuffer))

	for _, msg := range c.incomingBuffer {
		if processed[msg.MessageId] {
			continue
		}

		missingDeps := c.checkCausalDependencies(msg)
		if len(missingDeps) == 0 {
			// Release the lock while processing the message
			c.mutex.Unlock()
			err := c.processMessageWithoutBuffering(msg)
			c.mutex.Lock()

			if err != nil {
				remainingBuffer = append(remainingBuffer, msg)
			} else {
				processed[msg.MessageId] = true
			}
		} else {
			remainingBuffer = append(remainingBuffer, msg)
		}
	}

	c.incomingBuffer = remainingBuffer
}

func (c *Chat) processMessageWithoutBuffering(msg *pb.Message) error {
	if msg.Content != "" {
		c.ui.ChatMessage(int64(c.getLamportTimestamp()), msg.SenderId, msg.Content)
	}

	c.addToMessageHistory(msg)
	return nil
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
			// Process incoming buffer
			c.processBufferedMessages()

			// Resend unacknowledged messages from outgoing buffer
			c.checkUnacknowledgedMessages()
		}
	}
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

// Methods to handle disconnections
func (c *Chat) SetDisconnected(disconnected bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.isDisconnected = disconnected
}

func (c *Chat) IsDisconnected() bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.isDisconnected
}
