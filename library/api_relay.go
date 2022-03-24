package main

import (
	"C"
	"context"
	"encoding/json"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/status-im/go-waku/waku/v2/node"
	"github.com/status-im/go-waku/waku/v2/protocol"
	"github.com/status-im/go-waku/waku/v2/protocol/pb"
)
import (
	"sync"

	"github.com/status-im/go-waku/waku/v2/protocol/relay"
)

var subscriptions map[string]*relay.Subscription = make(map[string]*relay.Subscription)
var mutex sync.Mutex

//export waku_relay_enough_peers
// Determine if there are enough peers to publish a message on a topic. Use NULL
// to verify the number of peers in the default pubsub topic
func waku_relay_enough_peers(topic *C.char) *C.char {
	if wakuNode == nil {
		return makeJSONResponse(ErrWakuNodeNotReady)
	}

	topicToCheck := protocol.DefaultPubsubTopic().String()
	if topic != nil {
		topicToCheck = C.GoString(topic)
	}

	return prepareJSONResponse(wakuNode.Relay().EnoughPeersToPublishToTopic(topicToCheck), nil)
}

func relay_publish(msg pb.WakuMessage, pubsubTopic string, ms int) (string, error) {
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

	hash, err := wakuNode.Relay().PublishToTopic(ctx, &msg, pubsubTopic)
	return hexutil.Encode(hash), err
}

//export waku_relay_publish
// Publish a message using waku relay. Use NULL for topic to use the default pubsub topic
// If ms is greater than 0, the broadcast of the message must happen before the timeout
// (in milliseconds) is reached, or an error will be returned
func waku_relay_publish(messageJSON *C.char, topic *C.char, ms C.int) *C.char {
	var msg pb.WakuMessage
	err := json.Unmarshal([]byte(C.GoString(messageJSON)), &msg)
	if err != nil {
		return makeJSONResponse(err)
	}

	msg.Version = 0

	hash, err := relay_publish(msg, getTopic(topic), int(ms))
	return prepareJSONResponse(hash, err)
}

//export waku_relay_publish_enc_asymmetric
// Publish a message encrypted with a secp256k1 public key using waku relay. Use NULL for topic to use the default pubsub topic.
// publicKey must be a hex string prefixed with "0x" containing a valid secp256k1 public key.
// optionalSigningKey is an optional hex string prefixed with "0x" containing a valid secp256k1 private key for signing the message. Use NULL otherwise
// If ms is greater than 0, the broadcast of the message must happen before the timeout
// (in milliseconds) is reached, or an error will be returned. 
func waku_relay_publish_enc_asymmetric(messageJSON *C.char, topic *C.char, publicKey *C.char, optionalSigningKey *C.char, ms C.int) *C.char {
	var msg pb.WakuMessage
	err := json.Unmarshal([]byte(C.GoString(messageJSON)), &msg)
	if err != nil {
		return makeJSONResponse(err)
	}

	payload := node.Payload{
		Data: msg.Payload,
		Key: &node.KeyInfo{
			Kind: node.Asymmetric,
		},
	}

	keyBytes, err := hexutil.Decode(C.GoString(publicKey))
	if err != nil {
		return makeJSONResponse(err)
	}

	payload.Key.PubKey, err = unmarshalPubkey(keyBytes)
	if err != nil {
		return makeJSONResponse(err)
	}

	if optionalSigningKey != nil {
		signingKeyBytes, err := hexutil.Decode(C.GoString(optionalSigningKey))
		if err != nil {
			return makeJSONResponse(err)
		}

		payload.Key.PrivKey, err = crypto.ToECDSA(signingKeyBytes)
		if err != nil {
			return makeJSONResponse(err)
		}
	}

	msg.Version = 1
	msg.Payload, err = payload.Encode(1)
	if err != nil {
		return makeJSONResponse(err)
	}

	hash, err := relay_publish(msg, getTopic(topic), int(ms))

	return prepareJSONResponse(hash, err)
}

//export waku_relay_publish_enc_symmetric
// Publish a message encrypted with a 32 bytes symmetric key using waku relay. Use NULL for topic to use the default pubsub topic.
// publicKey must be a hex string prefixed with "0x" containing a 32 bytes symmetric key
// optionalSigningKey is an optional hex string prefixed with "0x" containing a valid secp256k1 private key for signing the message. Use NULL otherwise
// If ms is greater than 0, the broadcast of the message must happen before the timeout
// (in milliseconds) is reached, or an error will be returned. 
func waku_relay_publish_enc_symmetric(messageJSON *C.char, topic *C.char, symmetricKey *C.char, optionalSigningKey *C.char, ms C.int) *C.char {
	var msg pb.WakuMessage
	err := json.Unmarshal([]byte(C.GoString(messageJSON)), &msg)
	if err != nil {
		return makeJSONResponse(err)
	}

	payload := node.Payload{
		Data: msg.Payload,
		Key: &node.KeyInfo{
			Kind: node.Symmetric,
		},
	}

	payload.Key.SymKey, err = hexutil.Decode(C.GoString(symmetricKey))
	if err != nil {
		return makeJSONResponse(err)
	}

	if optionalSigningKey != nil {
		signingKeyBytes, err := hexutil.Decode(C.GoString(optionalSigningKey))
		if err != nil {
			return makeJSONResponse(err)
		}

		payload.Key.PrivKey, err = crypto.ToECDSA(signingKeyBytes)
		if err != nil {
			return makeJSONResponse(err)
		}
	}

	msg.Version = 1
	msg.Payload, err = payload.Encode(1)
	if err != nil {
		return makeJSONResponse(err)
	}

	hash, err := relay_publish(msg, getTopic(topic), int(ms))

	return prepareJSONResponse(hash, err)
}

//export waku_relay_subscribe
// Subscribe to a WakuRelay topic. Set the topic to NULL to subscribe
// to the default topic. Returns a json response containing the subscription ID
// or an error message. When a message is received, a "message" is emitted containing
// the message, pubsub topic, and nodeID in which the message was received
func waku_relay_subscribe(topic *C.char) *C.char {
	if wakuNode == nil {
		return makeJSONResponse(ErrWakuNodeNotReady)
	}

	topicToSubscribe := protocol.DefaultPubsubTopic().String()
	if topic != nil {
		topicToSubscribe = C.GoString(topic)
	}

	mutex.Lock()
	defer mutex.Unlock()

	subscription, ok := subscriptions[topicToSubscribe]
	if ok {
		return makeJSONResponse(nil)
	}

	subscription, err := wakuNode.Relay().SubscribeToTopic(context.Background(), topicToSubscribe)
	if err != nil {
		return makeJSONResponse(err)
	}

	subscriptions[topicToSubscribe] = subscription

	go func() {
		for envelope := range subscription.C {
			send("message", toSubscriptionMessage(envelope))
		}
	}()

	return makeJSONResponse(nil)
}

//export waku_relay_unsubscribe
// Closes the pubsub subscription to a pubsub topic. Existing subscriptions
// will not be closed, but they will stop receiving messages
func waku_relay_unsubscribe(topic *C.char) *C.char {
	if wakuNode == nil {
		return makeJSONResponse(ErrWakuNodeNotReady)
	}

	topicToUnsubscribe := protocol.DefaultPubsubTopic().String()
	if topic != nil {
		topicToUnsubscribe = C.GoString(topic)
	}

	mutex.Lock()
	defer mutex.Unlock()

	subscription, ok := subscriptions[topicToUnsubscribe]
	if ok {
		return makeJSONResponse(nil)
	}

	subscription.Unsubscribe()

	delete(subscriptions, topicToUnsubscribe)

	err := wakuNode.Relay().Unsubscribe(context.Background(), topicToUnsubscribe)
	if err != nil {
		return makeJSONResponse(err)
	}

	return makeJSONResponse(nil)
}
