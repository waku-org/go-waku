package common

import "github.com/status-im/go-waku/waku/v2/protocol"

type Envelope struct {
	msg  *protocol.WakuMessage
	size int
	hash []byte
}

func NewEnvelope(msg *protocol.WakuMessage, size int, hash []byte) *Envelope {
	return &Envelope{
		msg:  msg,
		size: size,
		hash: hash,
	}
}

func (e *Envelope) Message() *protocol.WakuMessage {
	return e.msg
}

func (e *Envelope) Hash() []byte {
	return e.hash
}

func (e *Envelope) Size() int {
	return e.size
}
