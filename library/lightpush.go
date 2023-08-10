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

	hash, err := wakuState.node.Lightpush().PublishToTopic(ctx, msg, pubsubTopic, lpOptions...)
	return hexutil.Encode(hash), err
}

// LightpushPublish is used to publish a WakuMessage in a pubsub topic using Lightpush protocol
func LightpushPublish(messageJSON string, topic string, peerID string, ms int) (string, error) {
	msg, err := wakuMessage(messageJSON)
	if err != nil {
		return "", err
	}

	return lightpushPublish(msg, getTopic(topic), peerID, ms)
}

// LightpushPublishEncodeAsymmetric is used to publish a WakuMessage in a pubsub topic using Lightpush protocol, and encrypting the message with some public key
func LightpushPublishEncodeAsymmetric(messageJSON string, topic string, peerID string, publicKey string, optionalSigningKey string, ms int) (string, error) {
	msg, err := wakuMessageAsymmetricEncoding(messageJSON, publicKey, optionalSigningKey)
	if err != nil {
		return "", err
	}

	return lightpushPublish(msg, getTopic(topic), peerID, ms)
}

// LightpushPublishEncodeSymmetric is used to publish a WakuMessage in a pubsub topic using Lightpush protocol, and encrypting the message with a symmetric key
func LightpushPublishEncodeSymmetric(messageJSON string, topic string, peerID string, symmetricKey string, optionalSigningKey string, ms int) (string, error) {
	msg, err := wakuMessageSymmetricEncoding(messageJSON, symmetricKey, optionalSigningKey)
	if err != nil {
		return "", err
	}

	return lightpushPublish(msg, getTopic(topic), peerID, ms)
}
