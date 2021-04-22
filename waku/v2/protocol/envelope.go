package protocol

import "github.com/status-im/go-waku/waku/v2/protocol/pb"

type Envelope struct {
	msg  *pb.WakuMessage
	size int
	hash []byte
}

func NewEnvelope(msg *pb.WakuMessage, size int, hash []byte) *Envelope {
	return &Envelope{
		msg:  msg,
		size: size,
		hash: hash,
	}
}

func (e *Envelope) Message() *pb.WakuMessage {
	return e.msg
}

func (e *Envelope) Hash() []byte {
	return e.hash
}

func (e *Envelope) Size() int {
	return e.size
}
