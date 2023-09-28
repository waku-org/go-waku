package library

import (
	"context"
	"time"

	"github.com/waku-org/go-waku/waku/v2/protocol/pb"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/waku-org/go-waku/waku/v2/protocol/lightpush"
)

func lightpushPublish(msg *pb.WakuMessage, pubsubTopic string, peerID string, ms int) (string, error) {
	if wakuState.node == nil {
		return "", errWakuNodeNotReady
	}

	var ctx context.Context
	var cancel context.CancelFunc

	if ms > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(int(ms))*time.Millisecond)
		defer cancel()
	} else {
		ctx = context.Background()
	}

	var lpOptions []lightpush.Option
	if peerID != "" {
		p, err := peer.Decode(peerID)
		if err != nil {
			return "", err
		}
		lpOptions = append(lpOptions, lightpush.WithPeer(p))
	} else {
		lpOptions = append(lpOptions, lightpush.WithAutomaticPeerSelection())
	}

	if pubsubTopic != "" {
		lpOptions = append(lpOptions, lightpush.WithPubSubTopic(pubsubTopic))
	}

	hash, err := wakuState.node.Lightpush().PublishToTopic(ctx, msg, lpOptions...)
	return hexutil.Encode(hash), err
}

// LightpushPublish is used to publish a WakuMessage in a pubsub topic using Lightpush protocol
func LightpushPublish(messageJSON string, pubsubTopic string, peerID string, ms int) (string, error) {
	msg, err := wakuMessage(messageJSON)
	if err != nil {
		return "", err
	}

	return lightpushPublish(msg, pubsubTopic, peerID, ms)
}
