package main

import (
	"C"

	"github.com/status-im/go-waku/waku/v2/protocol/pb"
)
import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/status-im/go-waku/waku/v2/protocol/lightpush"
)

func lightpushPublish(msg pb.WakuMessage, pubsubTopic string, peerID string, ms int) (string, error) {
	if wakuNode == nil {
		return "", ErrWakuNodeNotReady
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
		lpOptions = append(lpOptions, lightpush.WithAutomaticPeerSelection(wakuNode.Host()))
	}

	hash, err := wakuNode.Lightpush().PublishToTopic(ctx, &msg, pubsubTopic, lpOptions...)
	return hexutil.Encode(hash), err
}

//export waku_lightpush_publish
// Publish a message using waku lightpush. Use NULL for topic to use the default pubsub topic..
// peerID should contain the ID of a peer supporting the lightpush protocol. Use NULL to automatically select a node
// If ms is greater than 0, the broadcast of the message must happen before the timeout
// (in milliseconds) is reached, or an error will be returned
func waku_lightpush_publish(messageJSON *C.char, topic *C.char, peerID *C.char, ms C.int) *C.char {
	msg, err := wakuMessage(C.GoString(messageJSON))
	if err != nil {
		return makeJSONResponse(err)
	}

	hash, err := lightpushPublish(msg, getTopic(topic), C.GoString(peerID), int(ms))
	return prepareJSONResponse(hash, err)
}

//export waku_lightpush_publish_enc_asymmetric
// Publish a message encrypted with a secp256k1 public key using waku lightpush. Use NULL for topic to use the default pubsub topic.
// peerID should contain the ID of a peer supporting the lightpush protocol. Use NULL to automatically select a node
// publicKey must be a hex string prefixed with "0x" containing a valid secp256k1 public key.
// optionalSigningKey is an optional hex string prefixed with "0x" containing a valid secp256k1 private key for signing the message. Use NULL otherwise
// If ms is greater than 0, the broadcast of the message must happen before the timeout
// (in milliseconds) is reached, or an error will be returned.
func waku_lightpush_publish_enc_asymmetric(messageJSON *C.char, topic *C.char, peerID *C.char, publicKey *C.char, optionalSigningKey *C.char, ms C.int) *C.char {
	msg, err := wakuMessageAsymmetricEncoding(C.GoString(messageJSON), C.GoString(publicKey), C.GoString(optionalSigningKey))
	if err != nil {
		return makeJSONResponse(err)
	}

	hash, err := lightpushPublish(msg, getTopic(topic), C.GoString(peerID), int(ms))

	return prepareJSONResponse(hash, err)
}

//export waku_lightpush_publish_enc_symmetric
// Publish a message encrypted with a 32 bytes symmetric key using waku relay. Use NULL for topic to use the default pubsub topic.
// peerID should contain the ID of a peer supporting the lightpush protocol. Use NULL to automatically select a node
// publicKey must be a hex string prefixed with "0x" containing a 32 bytes symmetric key
// optionalSigningKey is an optional hex string prefixed with "0x" containing a valid secp256k1 private key for signing the message. Use NULL otherwise
// If ms is greater than 0, the broadcast of the message must happen before the timeout
// (in milliseconds) is reached, or an error will be returned.
func waku_lightpush_publish_enc_symmetric(messageJSON *C.char, topic *C.char, peerID *C.char, symmetricKey *C.char, optionalSigningKey *C.char, ms C.int) *C.char {
	msg, err := wakuMessageSymmetricEncoding(C.GoString(messageJSON), C.GoString(symmetricKey), C.GoString(optionalSigningKey))
	if err != nil {
		return makeJSONResponse(err)
	}

	hash, err := lightpushPublish(msg, getTopic(topic), C.GoString(peerID), int(ms))

	return prepareJSONResponse(hash, err)
}
