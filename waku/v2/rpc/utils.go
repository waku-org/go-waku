package rpc

import (
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
)

type RateLimitProof struct {
	Proof         []byte `json:"proof,omitempty"`
	MerkleRoot    []byte `json:"merkle_root,omitempty"`
	Epoch         []byte `json:"epoch,omitempty"`
	ShareX        []byte `json:"share_x,omitempty"`
	ShareY        []byte `json:"share_y,omitempty"`
	Nullifier     []byte `json:"nullifier,omitempty"`
	RlnIdentifier []byte `json:"rln_identifier,omitempty"`
}

type RPCWakuMessage struct {
	Payload        []byte          `json:"payload,omitempty"`
	ContentTopic   string          `json:"contentTopic,omitempty"`
	Version        uint32          `json:"version"`
	Timestamp      int64           `json:"timestamp,omitempty"`
	RateLimitProof *RateLimitProof `json:"rateLimitProof,omitempty"`
	Ephemeral      bool            `json:"ephemeral,omitempty"`
}

func ProtoToRPC(input *pb.WakuMessage) *RPCWakuMessage {
	if input == nil {
		return nil
	}

	rpcWakuMsg := &RPCWakuMessage{
		Payload:      input.Payload,
		ContentTopic: input.ContentTopic,
		Version:      input.Version,
		Timestamp:    input.Timestamp,
		Ephemeral:    input.Ephemeral,
	}

	if input.RateLimitProof != nil {
		rpcWakuMsg.RateLimitProof = &RateLimitProof{
			Proof:         input.RateLimitProof.Proof,
			MerkleRoot:    input.RateLimitProof.MerkleRoot,
			Epoch:         input.RateLimitProof.Epoch,
			ShareX:        input.RateLimitProof.ShareX,
			ShareY:        input.RateLimitProof.ShareY,
			Nullifier:     input.RateLimitProof.Nullifier,
			RlnIdentifier: input.RateLimitProof.RlnIdentifier,
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
		Ephemeral:    r.Ephemeral,
	}

	if r.RateLimitProof != nil {
		msg.RateLimitProof = &pb.RateLimitProof{
			Proof:         r.RateLimitProof.Proof,
			MerkleRoot:    r.RateLimitProof.MerkleRoot,
			Epoch:         r.RateLimitProof.Epoch,
			ShareX:        r.RateLimitProof.ShareX,
			ShareY:        r.RateLimitProof.ShareY,
			Nullifier:     r.RateLimitProof.Nullifier,
			RlnIdentifier: r.RateLimitProof.RlnIdentifier,
		}
	}

	return msg
}
