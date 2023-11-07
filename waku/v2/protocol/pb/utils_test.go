package pb

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/waku/v2/hash"
	"google.golang.org/protobuf/proto"
)

func TestHash(t *testing.T) {
	expected := []byte{0xa5, 0x91, 0xa6, 0xd4, 0xb, 0xf4, 0x20, 0x40, 0x4a, 0x1, 0x17, 0x33, 0xcf, 0xb7, 0xb1, 0x90, 0xd6, 0x2c, 0x65, 0xbf, 0xb, 0xcd, 0xa3, 0x2b, 0x57, 0xb2, 0x77, 0xd9, 0xad, 0x9f, 0x14, 0x6e}
	result := hash.SHA256([]byte("Hello World"))
	require.Equal(t, expected, result)
}

func TestEnvelopeHash(t *testing.T) {
	msg := new(WakuMessage)
	msg.ContentTopic = "Test"
	msg.Payload = []byte("Hello World")
	msg.Timestamp = proto.Int64(123456789123456789)
	msg.Version = proto.Uint32(1)

	expected := []byte{0xee, 0xcf, 0xf5, 0xb7, 0xdd, 0x54, 0x2d, 0x68, 0x9e, 0x7d, 0x64, 0xa3, 0xb8, 0x50, 0x8b, 0xba, 0xc, 0xf1, 0xac, 0xb6, 0xf7, 0x1c, 0x9f, 0xf2, 0x32, 0x7, 0x5b, 0xfd, 0x90, 0x5c, 0xe5, 0xa1}
	result := msg.Hash("test")
	require.Equal(t, expected, result)
}

func TestEmptyMeta(t *testing.T) {
	pubsubTopic := "/waku/2/default-waku/proto"
	msg := new(WakuMessage)
	msg.ContentTopic = "/waku/2/default-content/proto"
	msg.Payload = []byte("\x01\x02\x03\x04TEST\x05\x06\x07\x08")

	msg.Meta = []byte{}
	msg.Timestamp = proto.Int64(123456789123456789)
	msg.Version = proto.Uint32(1)

	messageHash := msg.Hash(pubsubTopic)

	require.Equal(t, "87619d05e563521d9126749b45bd4cc2430df0607e77e23572d874ed9c1aaa62", hex.EncodeToString(messageHash))
}

func Test13ByteMeta(t *testing.T) {
	pubsubTopic := "/waku/2/default-waku/proto"
	msg := new(WakuMessage)
	msg.ContentTopic = "/waku/2/default-content/proto"
	msg.Payload = []byte("\x01\x02\x03\x04TEST\x05\x06\x07\x08")
	msg.Meta = []byte("\x73\x75\x70\x65\x72\x2d\x73\x65\x63\x72\x65\x74")
	msg.Version = proto.Uint32(1)

	messageHash := msg.Hash(pubsubTopic)

	require.Equal(t, "4fdde1099c9f77f6dae8147b6b3179aba1fc8e14a7bf35203fc253ee479f135f", hex.EncodeToString(messageHash))
}

func TestZeroLenPayload(t *testing.T) {
	pubsubTopic := "/waku/2/default-waku/proto"
	msg := new(WakuMessage)
	msg.ContentTopic = "/waku/2/default-content/proto"
	msg.Payload = []byte{}
	msg.Meta = []byte("\x73\x75\x70\x65\x72\x2d\x73\x65\x63\x72\x65\x74")
	msg.Version = proto.Uint32(1)

	messageHash := msg.Hash(pubsubTopic)

	require.Equal(t, "e1a9596237dbe2cc8aaf4b838c46a7052df6bc0d42ba214b998a8bfdbe8487d6", hex.EncodeToString(messageHash))
}
