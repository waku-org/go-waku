package main

import (
	"chat2-reliable/pb"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bits-and-blooms/bloom/v3"
)

const (
	bloomFilterSize          = 10000
	bloomFilterFPRate        = 0.01
	bloomFilterWindow        = 1 * time.Hour
	bloomFilterCleanInterval = 30 * time.Minute
	bufferSweepInterval      = 5 * time.Second
	syncMessageInterval      = 30 * time.Second
	messageAckTimeout        = 60 * time.Second
	maxRetries               = 3
	retryBaseDelay           = 1 * time.Second
	maxRetryDelay            = 10 * time.Second
	ackTimeout               = 5 * time.Second
	maxResendAttempts        = 5
	resendBaseDelay          = 1 * time.Second
	maxResendDelay           = 30 * time.Second
)

func (c *Chat) initReliabilityProtocol() {
	c.wg.Add(4)
	c.setupMessageRequestHandler()

	go c.periodicBufferSweep()
	go c.periodicSyncMessage()
	go c.startBloomFilterCleaner()
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

type UnacknowledgedMessage struct {
	Message        *pb.Message
	SendTime       time.Time
	ResendAttempts int
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

func (c *Chat) SendMessage(line string) {
	c.incLamportTimestamp()

	bloomBytes, err := c.bloomFilter.MarshalBinary()
	if err != nil {
		c.ui.ErrorMessage(fmt.Errorf("failed to marshal bloom filter: %w", err))
		return
	}

	msg := &pb.Message{
		SenderId:         c.node.Host().ID().String(),
		MessageId:        generateUniqueID(),
		LamportTimestamp: c.getLamportTimestamp(),
		CausalHistory:    c.getRecentMessageIDs(10),
		ChannelId:        c.options.ContentTopic,
		BloomFilter:      bloomBytes,
		Content:          line,
	}

	unackMsg := UnacknowledgedMessage{
		Message:        msg,
		SendTime:       time.Now(),
		ResendAttempts: 0,
	}
	c.outgoingBuffer = append(c.outgoingBuffer, unackMsg)

	ctx, cancel := context.WithTimeout(c.ctx, messageAckTimeout)
	defer cancel()

	err = c.publish(ctx, msg)
	if err != nil {
		if err.Error() == "validation failed" {
			err = errors.New("message rate violation")
		}
		c.ui.ErrorMessage(err)
	} else {
		c.bloomFilter.Add(msg.MessageId)
		c.addToMessageHistory(msg)
		c.ui.ChatMessage(int64(c.getLamportTimestamp()), msg.SenderId, msg.Content)
	}
}

func (c *Chat) processReceivedMessage(msg *pb.Message) {
	// Check if the message is already in the bloom filter
	if c.bloomFilter.Test(msg.MessageId) {
		return
	}

	// Update bloom filter
	c.bloomFilter.Add(msg.MessageId)

	// Update Lamport timestamp
	c.updateLamportTimestamp(msg.LamportTimestamp)

	// Review ACK status of messages in the unacknowledged outgoing buffer
	c.reviewAckStatus(msg)

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

	remainingBuffer := make([]*pb.Message, 0, len(c.incomingBuffer))
	processedBuffer := make([]*pb.Message, 0)

	for _, msg := range c.incomingBuffer {
		missingDeps := c.checkCausalDependencies(msg)
		if len(missingDeps) == 0 {
			if msg.Content != "" {
				c.ui.ChatMessage(int64(c.getLamportTimestamp()), msg.SenderId, msg.Content)
				processedBuffer = append(processedBuffer, msg)
			}
		} else {
			remainingBuffer = append(remainingBuffer, msg)
		}
	}

	c.incomingBuffer = remainingBuffer
	c.mutex.Unlock()

	for _, msg := range processedBuffer {
		c.addToMessageHistory(msg)
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
		if err != nil {
			c.ui.ErrorMessage(fmt.Errorf("failed to unmarshal bloom filter: %w", err))
		} else {
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
		missedMsg, err := c.doRequestMissingMessageFromPeers(messageID)
		if err == nil {
			c.processReceivedMessage(missedMsg)
			return
		}

		// Exponential backoff
		delay := retryBaseDelay * time.Duration(1<<uint(retry))
		if delay > maxRetryDelay {
			delay = maxRetryDelay
		}
		time.Sleep(delay)
	}

	c.ui.ErrorMessage(fmt.Errorf("failed to retrieve missing message %s after %d attempts", messageID, maxRetries))
}

func (c *Chat) checkCausalDependencies(msg *pb.Message) []string {
	var missingDeps []string
	seenMessages := make(map[string]bool)

	for _, historicalMsg := range c.messageHistory {
		seenMessages[historicalMsg.MessageId] = true
	}

	for _, depID := range msg.CausalHistory {
		if !seenMessages[depID] {
			missingDeps = append(missingDeps, depID)
		}
	}
	return missingDeps
}

func (c *Chat) addToMessageHistory(msg *pb.Message) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Find the correct position to insert the new message
	insertIndex := len(c.messageHistory)
	for i, existingMsg := range c.messageHistory {
		if existingMsg.LamportTimestamp > msg.LamportTimestamp {
			insertIndex = i
			break
		} else if existingMsg.LamportTimestamp == msg.LamportTimestamp {
			// If timestamps are equal, use MessageId for deterministic ordering
			if existingMsg.MessageId > msg.MessageId {
				insertIndex = i
				break
			}
		}
	}

	// Insert the new message at the correct position
	if insertIndex == len(c.messageHistory) {
		c.messageHistory = append(c.messageHistory, msg)
	} else {
		c.messageHistory = append(c.messageHistory[:insertIndex+1], c.messageHistory[insertIndex:]...)
		c.messageHistory[insertIndex] = msg
	}

	// Trim the history if it exceeds the maximum size
	if len(c.messageHistory) > maxMessageHistory {
		c.messageHistory = c.messageHistory[len(c.messageHistory)-maxMessageHistory:]
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
				c.resendMessage(unackMsg.Message, unackMsg.ResendAttempts)
				c.outgoingBuffer[i].ResendAttempts++
				c.outgoingBuffer[i].SendTime = now
			} else {
				// Remove the message from the buffer after max attempts
				c.outgoingBuffer = append(c.outgoingBuffer[:i], c.outgoingBuffer[i+1:]...)
				i-- // Adjust index after removal
				c.ui.ErrorMessage(fmt.Errorf("message %s dropped: failed to be acknowledged after %d attempts", unackMsg.Message.Content, maxResendAttempts))
			}
		}
	}
}

func (c *Chat) resendMessage(msg *pb.Message, resendAttempts int) {
	go func() {
		delay := resendBaseDelay * time.Duration(1<<uint(resendAttempts))
		if delay > maxResendDelay {
			delay = maxResendDelay
		}
		select {
		case <-c.ctx.Done():
			return
		case <-time.After(delay):
			// do nothing
		}

		ctx, cancel := context.WithTimeout(c.ctx, ackTimeout)
		defer cancel()

		err := c.publish(ctx, msg)
		if err != nil {
			c.ui.ErrorMessage(fmt.Errorf("failed to resend message: %w", err))
		}
	}()
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
	c.lamportTSMutex.Lock()
	defer c.lamportTSMutex.Unlock()
	c.lamportTimestamp++
}

func (c *Chat) updateLamportTimestamp(msgTs int32) {
	c.lamportTSMutex.Lock()
	defer c.lamportTSMutex.Unlock()
	if msgTs > c.lamportTimestamp {
		c.lamportTimestamp = msgTs
	}
}

func (c *Chat) getLamportTimestamp() int32 {
	c.lamportTSMutex.Lock()
	defer c.lamportTSMutex.Unlock()
	return c.lamportTimestamp
}
