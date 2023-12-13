package pb

import (
	"encoding/binary"

	"github.com/waku-org/go-waku/waku/v2/hash"
)

// Hash calculates the hash of a waku message
func (msg *WakuMessage) Hash(pubsubTopic string) []byte {
	return hash.SHA256([]byte(pubsubTopic), msg.Payload, []byte(msg.ContentTopic), msg.Meta, toBytes(msg.GetTimestamp()))
}

func toBytes(i int64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(i))
	return b
}
