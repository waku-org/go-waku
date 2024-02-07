package storev3

import (
	"context"

	"github.com/libp2p/go-libp2p/core/peer"
	wpb "github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/storev3/pb"
)

// Result represents a valid response from a store node
type Result struct {
	started      bool
	Messages     []*wpb.WakuMessage
	store        *WakuStoreV3
	storeRequest *pb.StoreRequest
	cursor       []byte
	peerID       peer.ID
}

func (r *Result) Cursor() []byte {
	return r.cursor
}

func (r *Result) IsComplete() bool {
	return r.cursor == nil
}

func (r *Result) PeerID() peer.ID {
	return r.peerID
}

func (r *Result) Query() *pb.StoreRequest {
	return r.storeRequest
}

func (r *Result) Next(ctx context.Context) (bool, error) {
	if !r.started {
		r.started = true
		return len(r.Messages) != 0, nil
	}

	if r.IsComplete() {
		return false, nil
	}

	newResult, err := r.store.Next(ctx, r)
	if err != nil {
		return false, err
	}

	r.cursor = newResult.cursor
	r.Messages = newResult.Messages

	return true, nil
}

func (r *Result) GetMessages() []*wpb.WakuMessage {
	if !r.started {
		return nil
	}
	return r.Messages
}
