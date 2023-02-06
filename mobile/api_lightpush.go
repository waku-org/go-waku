package gowaku

import (
	"context"
	"time"

	"github.com/waku-org/go-waku/waku/v2/protocol/pb"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/waku-org/go-waku/waku/v2/protocol/lightpush"
)

func lightpushPublish(msg *pb.WakuMessage, pubsubTopic string, peerID string, ms int) (string, error) {
	if wakuNode == nil {
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

	var lpOptions []lightpush.LightPushOption
	if peerID != "" {
		p, err := peer.Decode(peerID)
		if err != nil {
			return "", err
		}
		lpOptions = append(lpOptions, lightpush.WithPeer(p))
	} else {
		lpOptions = append(lpOptions, lightpush.WithAutomaticPeerSelection())
	}

	hash, err := wakuNode.Lightpush().PublishToTopic(ctx, msg, pubsubTopic, lpOptions...)
	return hexutil.Encode(hash), err
}

func LightpushPublish(messageJSON string, topic string, peerID string, ms int) string {
	msg, err := wakuMessage(messageJSON)
	if err != nil {
		return MakeJSONResponse(err)
	}

	hash, err := lightpushPublish(msg, getTopic(topic), peerID, ms)
	return PrepareJSONResponse(hash, err)
}

func LightpushPublishEncodeAsymmetric(messageJSON string, topic string, peerID string, publicKey string, optionalSigningKey string, ms int) string {
	msg, err := wakuMessageAsymmetricEncoding(messageJSON, publicKey, optionalSigningKey)
	if err != nil {
		return MakeJSONResponse(err)
	}

	hash, err := lightpushPublish(msg, getTopic(topic), peerID, ms)

	return PrepareJSONResponse(hash, err)
}

func LightpushPublishEncodeSymmetric(messageJSON string, topic string, peerID string, symmetricKey string, optionalSigningKey string, ms int) string {
	msg, err := wakuMessageSymmetricEncoding(messageJSON, symmetricKey, optionalSigningKey)
	if err != nil {
		return MakeJSONResponse(err)
	}

	hash, err := lightpushPublish(msg, getTopic(topic), peerID, ms)

	return PrepareJSONResponse(hash, err)
}
