package rpc

import (
	"errors"

	"github.com/waku-org/go-waku/cmd/waku/server"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	rlnpb "github.com/waku-org/go-waku/waku/v2/protocol/rln/pb"

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
	Payload        server.Base64URLByte `json:"payload,omitempty"`
	ContentTopic   string               `json:"contentTopic,omitempty"`
	Version        uint32               `json:"version"`
	Timestamp      int64                `json:"timestamp,omitempty"`
	RateLimitProof *RateLimitProof      `json:"rateLimitProof,omitempty"`
	Ephemeral      bool                 `json:"ephemeral,omitempty"`
}

func ProtoToRPC(input *pb.WakuMessage) (*RPCWakuMessage, error) {
	if input == nil {
		return nil, nil
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	rpcWakuMsg := &RPCWakuMessage{
		Payload:      input.Payload,
		ContentTopic: input.ContentTopic,
		Version:      input.GetVersion(),
		Timestamp:    input.GetTimestamp(),
		Ephemeral:    input.GetEphemeral(),
	}

	if input.RateLimitProof != nil {
		rateLimitProof := &rlnpb.RateLimitProof{}
		err := proto.Unmarshal(input.RateLimitProof, rateLimitProof)
		if err != nil {
			return nil, err
		}

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
		return nil, errors.New("wakumessage is missing")
	}

	msg := &pb.WakuMessage{
		Payload:      r.Payload,
		ContentTopic: r.ContentTopic,
		Version:      proto.Uint32(r.Version),
		Timestamp:    proto.Int64(r.Timestamp),
		Ephemeral:    proto.Bool(r.Ephemeral),
	}

	if r.RateLimitProof != nil {
		rateLimitProof := &rlnpb.RateLimitProof{
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
