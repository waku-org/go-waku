package main

import (
	"sync"
	"time"

	"github.com/bits-and-blooms/bloom/v3"
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
