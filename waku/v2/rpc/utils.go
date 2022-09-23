package rpc

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/status-im/go-waku/waku/v2/protocol/pb"
	"github.com/status-im/go-waku/waku/v2/utils"
)

// HexBytes is marshalled to a hex string
type HexBytes []byte

// ByteArray is marshalled to a uint8 array
type ByteArray []byte

type RateLimitProof struct {
	Proof      HexBytes `json:"proof,omitempty"`
	MerkleRoot HexBytes `json:"merkle_root,omitempty"`
	Epoch      HexBytes `json:"epoch,omitempty"`
	ShareX     HexBytes `json:"share_x,omitempty"`
	ShareY     HexBytes `json:"share_y,omitempty"`
	Nullifier  HexBytes `json:"nullifier,omitempty"`
}

type RPCWakuMessage struct {
	Payload        ByteArray       `json:"payload,omitempty"`
	ContentTopic   string          `json:"contentTopic,omitempty"`
	Version        uint32          `json:"version"`
	Timestamp      int64           `json:"timestamp,omitempty"`
	RateLimitProof *RateLimitProof `json:"rateLimitProof,omitempty"`
}

type RPCWakuRelayMessage struct {
	Payload        HexBytes        `json:"payload,omitempty"`
	ContentTopic   string          `json:"contentTopic,omitempty"`
	Timestamp      int64           `json:"timestamp,omitempty"`
	RateLimitProof *RateLimitProof `json:"rateLimitProof,omitempty"`
	Version        uint32          `json:"version"`
}

func ProtoWakuMessageToRPCWakuMessage(input *pb.WakuMessage) *RPCWakuMessage {
	if input == nil {
		return nil
	}

	rpcWakuMsg := &RPCWakuMessage{
		Payload:      input.Payload,
		ContentTopic: input.ContentTopic,
		Version:      input.Version,
		Timestamp:    input.Timestamp,
	}

	if input.RateLimitProof != nil {
		rpcWakuMsg.RateLimitProof = &RateLimitProof{
			Proof:      input.RateLimitProof.Proof,
			MerkleRoot: input.RateLimitProof.MerkleRoot,
			Epoch:      input.RateLimitProof.Epoch,
			ShareX:     input.RateLimitProof.ShareX,
			ShareY:     input.RateLimitProof.ShareY,
			Nullifier:  input.RateLimitProof.Nullifier,
		}
	}

	return rpcWakuMsg
}

func (r *RPCWakuMessage) toProto() *pb.WakuMessage {
	if r == nil {
		return nil
	}

	msg := &pb.WakuMessage{
		Payload:      r.Payload,
		ContentTopic: r.ContentTopic,
		Version:      r.Version,
		Timestamp:    r.Timestamp,
	}

	if r.RateLimitProof != nil {
		msg.RateLimitProof = &pb.RateLimitProof{
			Proof:      r.RateLimitProof.Proof,
			MerkleRoot: r.RateLimitProof.MerkleRoot,
			Epoch:      r.RateLimitProof.Epoch,
			ShareX:     r.RateLimitProof.ShareX,
			ShareY:     r.RateLimitProof.ShareY,
			Nullifier:  r.RateLimitProof.Nullifier,
		}
	}

	return msg
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

	decoded, err := utils.DecodeHexString(hexString)
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

	rpcMsg := &RPCWakuRelayMessage{
		Payload:      input.Payload,
		ContentTopic: input.ContentTopic,
		Timestamp:    input.Timestamp,
	}

	if input.RateLimitProof != nil {
		rpcMsg.RateLimitProof = &RateLimitProof{
			Proof:      input.RateLimitProof.Proof,
			MerkleRoot: input.RateLimitProof.MerkleRoot,
			Epoch:      input.RateLimitProof.Epoch,
			ShareX:     input.RateLimitProof.ShareX,
			ShareY:     input.RateLimitProof.ShareY,
			Nullifier:  input.RateLimitProof.Nullifier,
		}
	}

	return rpcMsg
}

func (r *RPCWakuRelayMessage) toProto() *pb.WakuMessage {
	if r == nil {
		return nil
	}

	msg := &pb.WakuMessage{
		Payload:      r.Payload,
		ContentTopic: r.ContentTopic,
		Timestamp:    r.Timestamp,
		Version:      r.Version,
	}

	if r.RateLimitProof != nil {
		msg.RateLimitProof = &pb.RateLimitProof{
			Proof:      r.RateLimitProof.Proof,
			MerkleRoot: r.RateLimitProof.MerkleRoot,
			Epoch:      r.RateLimitProof.Epoch,
			ShareX:     r.RateLimitProof.ShareX,
			ShareY:     r.RateLimitProof.ShareY,
			Nullifier:  r.RateLimitProof.Nullifier,
		}
	}

	return msg
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

	decoded, err := utils.DecodeHexString(hexString)
	if err != nil {
		return err
	}

	*h = decoded

	return nil
}
