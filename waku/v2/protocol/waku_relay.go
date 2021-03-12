// Waku Relay module. Thin layer on top of GossipSub.
//
// See https://github.com/vacp2p/specs/blob/master/specs/waku/v2/waku-relay.md
// for spec.

package protocol

import (
	"context"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/protocol"
	libp2pProtocol "github.com/libp2p/go-libp2p-core/protocol"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

const WakuRelayCodec = libp2pProtocol.ID("/vac/waku/relay/2.0.0-beta2")

type WakuRelaySubRouter struct {
	*pubsub.GossipSubRouter
	p *pubsub.PubSub
}

func NewWakuRelaySub(ctx context.Context, h host.Host) (*pubsub.PubSub, error) {
	opts := []pubsub.Option{
		pubsub.WithMessageSignaturePolicy(pubsub.StrictNoSign),
	}
	gossipSub, err := pubsub.NewGossipSub(ctx, h, opts...)

	if err != nil {
		return nil, err
	}

	w := new(WakuRelaySubRouter)
	w.p = gossipSub
	return gossipSub, nil
}

func (ws *WakuRelaySubRouter) Protocols() []protocol.ID {
	return []libp2pProtocol.ID{WakuRelayCodec, pubsub.GossipSubID_v11, pubsub.GossipSubID_v10, pubsub.FloodSubID}
}
