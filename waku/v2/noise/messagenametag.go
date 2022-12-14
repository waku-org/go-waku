package noise

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
)

type MessageNametag = [MessageNametagLength]byte

const MessageNametagLength = 16
const MessageNametagBufferSize = 50

var (
	ErrNametagNotFound    = errors.New("message nametag not found in buffer")
	ErrNametagNotExpected = errors.New("message nametag is present in buffer but is not the next expected nametag. One or more messages were probably lost")
)

// Converts a sequence or array (arbitrary size) to a MessageNametag
func BytesToMessageNametag(input []byte) MessageNametag {
	var result MessageNametag
	copy(result[:], input)
	return result
}

type MessageNametagBuffer struct {
	buffer  []MessageNametag
	counter uint64
	secret  []byte
}

func NewMessageNametagBuffer(secret []byte) *MessageNametagBuffer {
	return &MessageNametagBuffer{
		secret: secret,
	}
}

// Initializes the empty Message nametag buffer. The n-th nametag is equal to HKDF( secret || n )
func (m *MessageNametagBuffer) Init() {
	// We default the counter and buffer fields
	m.counter = 0
	m.buffer = make([]MessageNametag, MessageNametagBufferSize)
	if len(m.secret) != 0 {
		for i := range m.buffer {
			counterBytesLE := make([]byte, 8)
			binary.LittleEndian.PutUint64(counterBytesLE, m.counter)
			toHash := []byte{}
			toHash = append(toHash, m.secret...)
			toHash = append(toHash, counterBytesLE...)
			d := sha256.Sum256(toHash)
			m.buffer[i] = BytesToMessageNametag(d[:])
			m.counter++
		}
	}
}

func (m *MessageNametagBuffer) Pop() MessageNametag {
	// Note that if the input MessageNametagBuffer is set to default, an all 0 messageNametag is returned
	if len(m.buffer) == 0 {
		var m MessageNametag
		return m
	} else {
		messageNametag := m.buffer[0]
		m.Delete(1)
		return messageNametag
	}
}

// Checks if the input messageNametag is contained in the input MessageNametagBuffer
func (m *MessageNametagBuffer) CheckNametag(messageNametag MessageNametag) error {
	if len(m.buffer) != MessageNametagBufferSize {
		return nil
	}

	index := -1
	for i, x := range m.buffer {
		if bytes.Equal(x[:], messageNametag[:]) {
			index = i
			break
		}
	}

	if index == -1 {
		return ErrNametagNotFound
	} else if index > 0 {
		return ErrNametagNotExpected
	}

	// index is 0, hence the read message tag is the next expected one
	return nil
}

func rotateLeft(elems []MessageNametag, k int) []MessageNametag {
	if k < 0 || len(elems) == 0 {
		return elems
	}
	r := len(elems) - k%len(elems)

	result := elems[r:]
	result = append(result, elems[:r]...)

	return result
}

// Deletes the first n elements in buffer and appends n new ones
func (m *MessageNametagBuffer) Delete(n int) {
	if n <= 0 {
		return
	}

	// We ensure n is at most MessageNametagBufferSize (the buffer will be fully replaced)
	if n > MessageNametagBufferSize {
		n = MessageNametagBufferSize
	}

	// We update the last n values in the array if a secret is set
	// Note that if the input MessageNametagBuffer is set to default, nothing is done here
	if len(m.secret) != 0 {
		m.buffer = rotateLeft(m.buffer, n)
		for i := 0; i < n; i++ {
			counterBytesLE := make([]byte, 8)
			binary.LittleEndian.PutUint64(counterBytesLE, m.counter)
			toHash := []byte{}
			toHash = append(toHash, m.secret...)
			toHash = append(toHash, counterBytesLE...)
			d := sha256.Sum256(toHash)
			m.buffer[len(m.buffer)-n+i] = BytesToMessageNametag(d[:])
			m.counter++
		}
	}
}
