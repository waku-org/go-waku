package rpc

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/status-im/go-waku/waku/v2/protocol/pb"
)

// HexBytes is marshalled to a hex string
type HexBytes []byte

// ByteArray is marshalled to a uint8 array
type ByteArray []byte

type RPCWakuMessage struct {
	Payload      ByteArray `json:"payload,omitempty"`
	ContentTopic string    `json:"contentTopic,omitempty"`
	Version      uint32    `json:"version"`
	Timestamp    int64     `json:"timestamp,omitempty"`
	Proof        HexBytes  `json:"proof,omitempty"`
}

type RPCWakuRelayMessage struct {
	Payload      HexBytes `json:"payload,omitempty"`
	ContentTopic string   `json:"contentTopic,omitempty"`
	Timestamp    int64    `json:"timestamp,omitempty"`
	Proof        HexBytes `json:"proof,omitempty"`
	Version      uint32   `json:"version"`
}

func ProtoWakuMessageToRPCWakuMessage(input *pb.WakuMessage) *RPCWakuMessage {
	if input == nil {
		return nil
	}

	return &RPCWakuMessage{
		Payload:      input.Payload,
		ContentTopic: input.ContentTopic,
		Version:      input.Version,
		Timestamp:    input.Timestamp,
		Proof:        input.Proof,
	}
}

func (r *RPCWakuMessage) toProto() *pb.WakuMessage {
	if r == nil {
		return nil
	}

	return &pb.WakuMessage{
		Payload:      r.Payload,
		ContentTopic: r.ContentTopic,
		Version:      r.Version,
		Timestamp:    r.Timestamp,
		Proof:        r.Proof,
	}
}

func (u HexBytes) MarshalJSON() ([]byte, error) {
	var result string
	if u == nil {
		result = "null"
	} else {
		result = strings.Join(strings.Fields(fmt.Sprintf("%d", u)), ",")
	}
	return []byte(result), nil
}

func (h *HexBytes) UnmarshalText(b []byte) error {
	hexString := ""
	if b != nil {
		hexString = string(b)
	}

	decoded, err := hex.DecodeString(hexString)
	if err != nil {
		return err
	}

	*h = decoded

	return nil
}

func ProtoWakuMessageToRPCWakuRelayMessage(input *pb.WakuMessage) *RPCWakuRelayMessage {
	if input == nil {
		return nil
	}

	return &RPCWakuRelayMessage{
		Payload:      input.Payload,
		ContentTopic: input.ContentTopic,
		Timestamp:    input.Timestamp,
		Proof:        input.Proof,
	}
}

func (r *RPCWakuRelayMessage) toProto() *pb.WakuMessage {
	if r == nil {
		return nil
	}

	return &pb.WakuMessage{
		Payload:      r.Payload,
		ContentTopic: r.ContentTopic,
		Timestamp:    r.Timestamp,
		Proof:        r.Proof,
	}
}

func (h ByteArray) MarshalText() ([]byte, error) {
	if h == nil {
		return []byte{}, nil
	}

	return []byte(hex.EncodeToString(h)), nil
}

func (h *ByteArray) UnmarshalText(b []byte) error {
	hexString := ""
	if b != nil {
		hexString = string(b)
	}

	decoded, err := hex.DecodeString(hexString)
	if err != nil {
		return err
	}

	*h = decoded

	return nil
}
