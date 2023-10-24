package rpc

import (
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"google.golang.org/protobuf/proto"
)

type RateLimitProof struct {
	Proof         Base64URLByte `json:"proof,omitempty"`
	MerkleRoot    Base64URLByte `json:"merkle_root,omitempty"`
	Epoch         Base64URLByte `json:"epoch,omitempty"`
	ShareX        Base64URLByte `json:"share_x,omitempty"`
	ShareY        Base64URLByte `json:"share_y,omitempty"`
	Nullifier     Base64URLByte `json:"nullifier,omitempty"`
	RlnIdentifier Base64URLByte `json:"rln_identifier,omitempty"`
}

type RPCWakuMessage struct {
	Payload        Base64URLByte   `json:"payload,omitempty"`
	ContentTopic   string          `json:"contentTopic,omitempty"`
	Version        uint32          `json:"version"`
	Timestamp      int64           `json:"timestamp,omitempty"`
	RateLimitProof *RateLimitProof `json:"rateLimitProof,omitempty"`
	Ephemeral      bool            `json:"ephemeral,omitempty"`
}

func ProtoToRPC(input *pb.WakuMessage) (*RPCWakuMessage, error) {
	if input == nil {
		return nil, nil
	}

	rpcWakuMsg := &RPCWakuMessage{
		Payload:      input.Payload,
		ContentTopic: input.ContentTopic,
		Version:      input.Version,
		Timestamp:    input.Timestamp,
		Ephemeral:    input.Ephemeral,
	}

	rateLimitProof := &pb.RateLimitProof{}
	err := proto.Unmarshal(input.RateLimitProof, rateLimitProof)
	if err != nil {
		return nil, err
	}

	if input.RateLimitProof != nil {
		rpcWakuMsg.RateLimitProof = &RateLimitProof{
			Proof:         rateLimitProof.Proof,
			MerkleRoot:    rateLimitProof.MerkleRoot,
			Epoch:         rateLimitProof.Epoch,
			ShareX:        rateLimitProof.ShareX,
			ShareY:        rateLimitProof.ShareY,
			Nullifier:     rateLimitProof.Nullifier,
			RlnIdentifier: rateLimitProof.RlnIdentifier,
		}
	}

	return rpcWakuMsg, nil
}

func (r *RPCWakuMessage) toProto() (*pb.WakuMessage, error) {
	if r == nil {
		return nil, nil
	}

	msg := &pb.WakuMessage{
		Payload:      r.Payload,
		ContentTopic: r.ContentTopic,
		Version:      r.Version,
		Timestamp:    r.Timestamp,
		Ephemeral:    r.Ephemeral,
	}

	if r.RateLimitProof != nil {
		rateLimitProof := &pb.RateLimitProof{
			Proof:         r.RateLimitProof.Proof,
			MerkleRoot:    r.RateLimitProof.MerkleRoot,
			Epoch:         r.RateLimitProof.Epoch,
			ShareX:        r.RateLimitProof.ShareX,
			ShareY:        r.RateLimitProof.ShareY,
			Nullifier:     r.RateLimitProof.Nullifier,
			RlnIdentifier: r.RateLimitProof.RlnIdentifier,
		}

		b, err := proto.Marshal(rateLimitProof)
		if err != nil {
			return nil, err
		}

		msg.RateLimitProof = b
	}

	return msg, nil
}
