package main

/*
#include <stdlib.h>
#include <stddef.h>

typedef struct {
  size_t len;
  char* data;
} ByteArray;

#define SYMMETRIC "Symmetric"
#define ASYMMETRIC "Asymmetric"
#define NONE "None"
*/
import "C"
import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"time"
	"unsafe"

	"sync"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p-core/peer"
	p2pproto "github.com/libp2p/go-libp2p-core/protocol"
	"github.com/multiformats/go-multiaddr"
	"github.com/status-im/go-waku/waku/v2/node"
	"github.com/status-im/go-waku/waku/v2/protocol"
	"github.com/status-im/go-waku/waku/v2/protocol/pb"
	"github.com/status-im/go-waku/waku/v2/protocol/relay"
)

var nodes map[int]*node.WakuNode = make(map[int]*node.WakuNode)
var subscriptions map[string]*relay.Subscription = make(map[string]*relay.Subscription)
var mutex sync.Mutex

var ErrWakuNodeNotReady = errors.New("go-waku not initialized")

func randomHex(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func main() {}

type WakuConfig struct {
	Host              *string `json:"host,omitempty"`
	Port              *int    `json:"port,omitempty"`
	AdvertiseAddress  *string `json:"advertiseAddr,omitempty"`
	NodeKey           *string `json:"nodeKey,omitempty"`
	KeepAliveInterval *int    `json:"keepAliveInterval,omitempty"`
	EnableRelay       *bool   `json:"relay"`
	MinPeersToPublish *int    `json:"minPeersToPublish"`
}

var DefaultHost = "0.0.0.0"
var DefaultPort = 60000
var DefaultKeepAliveInterval = 20
var DefaultEnableRelay = true
var DefaultMinPeersToPublish = 0

func getConfig(configJSON *C.char) (WakuConfig, error) {
	var config WakuConfig
	if configJSON != nil {
		err := json.Unmarshal([]byte(C.GoString(configJSON)), &config)
		if err != nil {
			return WakuConfig{}, err
		}
	}

	if config.Host == nil {
		config.Host = &DefaultHost
	}

	if config.EnableRelay == nil {
		config.EnableRelay = &DefaultEnableRelay
	}

	if config.Host == nil {
		config.Host = &DefaultHost
	}

	if config.Port == nil {
		config.Port = &DefaultPort
	}

	if config.KeepAliveInterval == nil {
		config.KeepAliveInterval = &DefaultKeepAliveInterval
	}

	if config.MinPeersToPublish == nil {
		config.MinPeersToPublish = &DefaultMinPeersToPublish
	}

	return config, nil
}

//export gowaku_new
// Initialize a waku node. Receives a JSON string containing the configuration
// for the node. It can be NULL. Example configuration:
// ```
// {"host": "0.0.0.0", "port": 60000, "advertiseAddr": "1.2.3.4", "nodeKey": "0x123...567", "keepAliveInterval": 20, "relay": true}
// ```
// All keys are optional. If not specified a default value will be set:
// - host: IP address. Default 0.0.0.0
// - port: TCP port to listen. Default 60000. Use 0 for random
// - advertiseAddr: External IP
// - nodeKey: secp256k1 private key. Default random
// - keepAliveInterval: interval in seconds to ping all peers
// - relay: Enable WakuRelay. Default `true`
// This function will return a nodeID which should be used in all calls from this API that require
// interacting with the node.
func gowaku_new(configJSON *C.char) *C.char {
	config, err := getConfig(configJSON)
	if err != nil {
		return makeJSONResponse(err)
	}

	hostAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", *config.Host, *config.Port))
	if err != nil {
		return makeJSONResponse(err)
	}

	var prvKey *ecdsa.PrivateKey
	if config.NodeKey != nil {
		prvKey, err = crypto.HexToECDSA(*config.NodeKey)
		if err != nil {
			return makeJSONResponse(err)
		}
	} else {
		key, err := randomHex(32)
		if err != nil {
			return makeJSONResponse(err)
		}
		prvKey, err = crypto.HexToECDSA(key)
		if err != nil {
			return makeJSONResponse(err)
		}
	}

	opts := []node.WakuNodeOption{
		node.WithPrivateKey(prvKey),
		node.WithHostAddress(hostAddr),
		node.WithKeepAlive(time.Duration(*config.KeepAliveInterval) * time.Second),
	}

	if *config.EnableRelay {
		opts = append(opts, node.WithWakuRelayAndMinPeers(*config.MinPeersToPublish))
	}

	ctx := context.Background()
	wakuNode, err := node.New(ctx, opts...)

	if err != nil {
		return makeJSONResponse(err)
	}

	mutex.Lock()
	defer mutex.Unlock()

	id := len(nodes) + 1
	nodes[id] = wakuNode

	return prepareJSONResponse(id, nil)
}

//export gowaku_start
// Starts the waku node
func gowaku_start(nodeID C.int) *C.char {
	mutex.Lock()
	defer mutex.Unlock()
	wakuNode, ok := nodes[int(nodeID)]
	if !ok || wakuNode == nil {
		return makeJSONResponse(ErrWakuNodeNotReady)
	}

	if err := wakuNode.Start(); err != nil {
		return makeJSONResponse(err)
	}

	return makeJSONResponse(nil)
}

//export gowaku_stop
// Stops a waku node
func gowaku_stop(nodeID C.int) *C.char {
	mutex.Lock()
	defer mutex.Unlock()
	wakuNode, ok := nodes[int(nodeID)]
	if !ok || wakuNode == nil {
		return makeJSONResponse(ErrWakuNodeNotReady)
	}

	wakuNode.Stop()
	nodes[int(nodeID)] = nil

	return makeJSONResponse(nil)
}

//export gowaku_peerid
// Obtain the peer ID of the waku node
func gowaku_peerid(nodeID C.int) *C.char {
	mutex.Lock()
	defer mutex.Unlock()
	wakuNode, ok := nodes[int(nodeID)]
	if !ok || wakuNode == nil {
		return makeJSONResponse(ErrWakuNodeNotReady)
	}

	return prepareJSONResponse(wakuNode.ID(), nil)
}

//export gowaku_listen_addresses
// Obtain the multiaddresses the wakunode is listening to
func gowaku_listen_addresses(nodeID C.int) *C.char {
	mutex.Lock()
	defer mutex.Unlock()
	wakuNode, ok := nodes[int(nodeID)]
	if !ok || wakuNode == nil {
		return makeJSONResponse(ErrWakuNodeNotReady)
	}

	addrs, err := json.Marshal(wakuNode.ListenAddresses())
	return prepareJSONResponse(addrs, err)
}

//export gowaku_add_peer
// Add node multiaddress and protocol to the wakunode peerstore
func gowaku_add_peer(nodeID C.int, address *C.char, protocolID *C.char) *C.char {
	mutex.Lock()
	defer mutex.Unlock()
	wakuNode, ok := nodes[int(nodeID)]
	if !ok || wakuNode == nil {
		return makeJSONResponse(ErrWakuNodeNotReady)
	}

	ma, err := multiaddr.NewMultiaddr(C.GoString(address))
	if err != nil {
		return makeJSONResponse(err)
	}

	peerID, err := wakuNode.AddPeer(ma, p2pproto.ID(C.GoString(protocolID)))
	return prepareJSONResponse(peerID, err)
}

//export gowaku_dial_peer
// Dial peer at multiaddress. if ms > 0, cancel the function execution if it takes longer than N milliseconds
func gowaku_dial_peer(nodeID C.int, address *C.char, ms C.int) *C.char {
	mutex.Lock()
	defer mutex.Unlock()
	wakuNode, ok := nodes[int(nodeID)]
	if !ok || wakuNode == nil {
		return makeJSONResponse(ErrWakuNodeNotReady)
	}

	var ctx context.Context
	var cancel context.CancelFunc

	if ms > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(int(ms))*time.Millisecond)
		defer cancel()
	} else {
		ctx = context.Background()
	}

	err := wakuNode.DialPeer(ctx, C.GoString(address))
	return makeJSONResponse(err)
}

//export gowaku_dial_peerid
// Dial known peer by peerID. if ms > 0, cancel the function execution if it takes longer than N milliseconds
func gowaku_dial_peerid(nodeID C.int, peerID *C.char, ms C.int) *C.char {
	mutex.Lock()
	defer mutex.Unlock()
	wakuNode, ok := nodes[int(nodeID)]
	if !ok || wakuNode == nil {
		return makeJSONResponse(ErrWakuNodeNotReady)
	}

	var ctx context.Context
	var cancel context.CancelFunc

	pID, err := peer.Decode(C.GoString(peerID))
	if err != nil {
		return makeJSONResponse(err)
	}

	if ms > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(int(ms))*time.Millisecond)
		defer cancel()
	} else {
		ctx = context.Background()
	}

	err = wakuNode.DialPeerByID(ctx, pID)
	return makeJSONResponse(err)
}

//export gowaku_close_peer
// Close connection to peer at multiaddress
func gowaku_close_peer(nodeID C.int, address *C.char) *C.char {
	mutex.Lock()
	defer mutex.Unlock()
	wakuNode, ok := nodes[int(nodeID)]
	if !ok || wakuNode == nil {
		return makeJSONResponse(ErrWakuNodeNotReady)
	}

	err := wakuNode.ClosePeerByAddress(C.GoString(address))
	return makeJSONResponse(err)
}

//export gowaku_close_peerid
// Close connection to a known peer by peerID
func gowaku_close_peerid(nodeID C.int, peerID *C.char) *C.char {
	mutex.Lock()
	defer mutex.Unlock()
	wakuNode, ok := nodes[int(nodeID)]
	if !ok || wakuNode == nil {
		return makeJSONResponse(ErrWakuNodeNotReady)
	}

	pID, err := peer.Decode(C.GoString(peerID))
	if err != nil {
		return makeJSONResponse(err)
	}

	err = wakuNode.ClosePeerById(pID)
	return makeJSONResponse(err)
}

//export gowaku_peer_cnt
// Get number of connected peers
func gowaku_peer_cnt(nodeID C.int) *C.char {
	mutex.Lock()
	defer mutex.Unlock()
	wakuNode, ok := nodes[int(nodeID)]
	if !ok || wakuNode == nil {
		return makeJSONResponse(ErrWakuNodeNotReady)
	}

	return prepareJSONResponse(wakuNode.PeerCount(), nil)
}

//export gowaku_content_topic
// Create a content topic string according to RFC 23
func gowaku_content_topic(applicationName *C.char, applicationVersion C.uint, contentTopicName *C.char, encoding *C.char) *C.char {
	return C.CString(protocol.NewContentTopic(C.GoString(applicationName), uint(applicationVersion), C.GoString(contentTopicName), C.GoString(encoding)).String())
}

//export gowaku_pubsub_topic
// Create a pubsub topic string according to RFC 23
func gowaku_pubsub_topic(name *C.char, encoding *C.char) *C.char {
	return prepareJSONResponse(protocol.NewPubsubTopic(C.GoString(name), C.GoString(encoding)).String(), nil)
}

//export gowaku_default_pubsub_topic
// Get the default pubsub topic used in waku2: /waku/2/default-waku/proto
func gowaku_default_pubsub_topic() *C.char {
	return C.CString(protocol.DefaultPubsubTopic().String())
}

func publish(nodeID int, message string, pubsubTopic string, ms int) (string, error) {
	mutex.Lock()
	defer mutex.Unlock()
	wakuNode, ok := nodes[nodeID]
	if !ok || wakuNode == nil {
		return "", ErrWakuNodeNotReady
	}

	var msg pb.WakuMessage
	err := json.Unmarshal([]byte(message), &msg)
	if err != nil {
		return "", err
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

//export gowaku_relay_publish
// Publish a message using waku relay. Use NULL for topic to use the default pubsub topic
// If ms is greater than 0, the broadcast of the message must happen before the timeout
// (in milliseconds) is reached, or an error will be returned
func gowaku_relay_publish(nodeID C.int, messageJSON *C.char, topic *C.char, ms C.int) *C.char {
	topicToPublish := ""
	if topic != nil {
		topicToPublish = C.GoString(topic)
	} else {
		topicToPublish = protocol.DefaultPubsubTopic().String()
	}

	hash, err := publish(int(nodeID), C.GoString(messageJSON), topicToPublish, int(ms))
	return prepareJSONResponse(hash, err)
}

//export gowaku_enough_peers
// Determine if there are enough peers to publish a message on a topic. Use NULL
// to verify the number of peers in the default pubsub topic
func gowaku_enough_peers(nodeID C.int, topic *C.char) *C.char {
	mutex.Lock()
	defer mutex.Unlock()
	wakuNode, ok := nodes[int(nodeID)]
	if !ok || wakuNode == nil {
		return makeJSONResponse(ErrWakuNodeNotReady)
	}

	topicToCheck := protocol.DefaultPubsubTopic().String()
	if topic != nil {
		topicToCheck = C.GoString(topic)
	}

	return prepareJSONResponse(wakuNode.Relay().EnoughPeersToPublishToTopic(topicToCheck), nil)
}

//export gowaku_set_event_callback
// Register callback to act as signal handler and receive application signal
// (in JSON) which are used o react to asyncronous events in waku. The function
// signature for the callback should be `void myCallback(char* signalJSON)`
func gowaku_set_event_callback(cb unsafe.Pointer) {
	setEventCallback(cb)
}

type SubscriptionMsg struct {
	MessageID      string          `json:"messageID"`
	SubscriptionID string          `json:"subscriptionID"`
	PubsubTopic    string          `json:"pubsubTopic"`
	Message        *pb.WakuMessage `json:"wakuMessage"`
}

func toSubscriptionMessage(subsID string, msg *protocol.Envelope) *SubscriptionMsg {
	return &SubscriptionMsg{
		SubscriptionID: subsID,
		MessageID:      hexutil.Encode(msg.Hash()),
		PubsubTopic:    msg.PubsubTopic(),
		Message:        msg.Message(),
	}
}

//export gowaku_relay_subscribe
// Subscribe to a WakuRelay topic. Set the topic to NULL to subscribe
// to the default topic. Returns a json response containing the subscription ID
// or an error message. When a message is received, a "message" is emitted containing
// the message, pubsub topic, and nodeID in which the message was received
func gowaku_relay_subscribe(nodeID C.int, topic *C.char) *C.char {
	mutex.Lock()
	defer mutex.Unlock()
	wakuNode, ok := nodes[int(nodeID)]
	if !ok || wakuNode == nil {
		return makeJSONResponse(ErrWakuNodeNotReady)
	}

	topicToSubscribe := protocol.DefaultPubsubTopic().String()
	if topic != nil {
		topicToSubscribe = C.GoString(topic)
	}

	subscription, err := wakuNode.Relay().SubscribeToTopic(context.Background(), topicToSubscribe)
	if err != nil {
		return makeJSONResponse(err)
	}

	subsID := uuid.New().String()
	subscriptions[subsID] = subscription

	go func() {
		for envelope := range subscription.C {
			send(int(nodeID), "message", toSubscriptionMessage(subsID, envelope))
		}
	}()

	return prepareJSONResponse(subsID, nil)
}

//export gowaku_relay_unsubscribe_from_topic
// Closes the pubsub subscription to a pubsub topic. Existing subscriptions
// will not be closed, but they will stop receiving messages
func gowaku_relay_unsubscribe_from_topic(nodeID C.int, topic *C.char) *C.char {
	mutex.Lock()
	defer mutex.Unlock()
	wakuNode, ok := nodes[int(nodeID)]
	if !ok || wakuNode == nil {
		return makeJSONResponse(ErrWakuNodeNotReady)
	}

	topicToUnsubscribe := protocol.DefaultPubsubTopic().String()
	if topic != nil {
		topicToUnsubscribe = C.GoString(topic)
	}

	err := wakuNode.Relay().Unsubscribe(context.Background(), topicToUnsubscribe)
	if err != nil {
		return makeJSONResponse(err)
	}

	return makeJSONResponse(nil)
}

//export gowaku_relay_close_subscription
// Closes a waku relay subscription
func gowaku_relay_close_subscription(nodeID C.int, subsID *C.char) *C.char {
	mutex.Lock()
	defer mutex.Unlock()
	wakuNode, ok := nodes[int(nodeID)]
	if !ok || wakuNode == nil {
		return makeJSONResponse(ErrWakuNodeNotReady)
	}

	subscription, ok := subscriptions[C.GoString(subsID)]
	if !ok {
		return makeJSONResponse(errors.New("Subscription does not exist"))
	}

	subscription.Unsubscribe()

	delete(subscriptions, C.GoString(subsID))

	return makeJSONResponse(nil)
}

//export gowaku_peers
// Retrieve the list of peers known by the waku node
func gowaku_peers(nodeID C.int) *C.char {
	mutex.Lock()
	defer mutex.Unlock()
	wakuNode, ok := nodes[int(nodeID)]
	if !ok || wakuNode == nil {
		return makeJSONResponse(ErrWakuNodeNotReady)
	}

	peers, err := wakuNode.Peers()
	return prepareJSONResponse(peers, err)
}

func unmarshalPubkey(pub []byte) (*ecdsa.PublicKey, error) {
	x, y := elliptic.Unmarshal(secp256k1.S256(), pub)
	if x == nil {
		return nil, errors.New("invalid public key")
	}
	return &ecdsa.PublicKey{Curve: secp256k1.S256(), X: x, Y: y}, nil
}

//export gowaku_encode_data
// Encode a byte array. `keyType` defines the type of key to use: `NONE`,
// `ASYMMETRIC` and `SYMMETRIC`. `version` is used to define the type of
// payload encryption:
// When `version` is 0
// - No encryption is used
// When `version` is 1
// - If using `ASYMMETRIC` encoding, `key` must contain a secp256k1 public key
//   to encrypt the data with,
// - If using `SYMMETRIC` encoding, `key` must contain a 32 bytes symmetric key.
// The `signingKey` can contain an optional secp256k1 private key to sign the
// encoded message, otherwise NULL can be used.
func gowaku_encode_data(data *C.char, keyType *C.char, key *C.char, signingKey *C.char, version C.int) *C.char {
	keyInfo := &node.KeyInfo{
		Kind: node.KeyKind(C.GoString(keyType)),
	}

	keyBytes, err := hexutil.Decode(C.GoString(key))
	if err != nil {
		return makeJSONResponse(err)
	}

	if signingKey != nil {
		signingKeyBytes, err := hexutil.Decode(C.GoString(signingKey))
		if err != nil {
			return makeJSONResponse(err)
		}

		privK, err := crypto.ToECDSA(signingKeyBytes)
		if err != nil {
			return makeJSONResponse(err)
		}
		keyInfo.PrivKey = privK
	}

	switch keyInfo.Kind {
	case node.Symmetric:
		keyInfo.SymKey = keyBytes
	case node.Asymmetric:
		pubK, err := unmarshalPubkey(keyBytes)
		if err != nil {
			return makeJSONResponse(err)
		}
		keyInfo.PubKey = *pubK
	}

	b, err := base64.StdEncoding.DecodeString(C.GoString(data))
	if err != nil {
		return makeJSONResponse(err)
	}

	payload := node.Payload{
		Data: b,
		Key:  keyInfo,
	}

	response, err := payload.Encode(uint32(version))
	return prepareJSONResponse(response, err)
}

//export gowaku_decode_data
// Decode a byte array. `keyType` defines the type of key used: `NONE`,
// `ASYMMETRIC` and `SYMMETRIC`. `version` is used to define the type of
// encryption that was used in the payload:
// When `version` is 0
// - No encryption was used. It will return the original message payload
// When `version` is 1
// - If using `ASYMMETRIC` encoding, `key` must contain a secp256k1 public key
//   to decrypt the data with,
// - If using `SYMMETRIC` encoding, `key` must contain a 32 bytes symmetric key.
func gowaku_decode_data(data *C.char, keyType *C.char, key *C.char, version C.int) *C.char {
	b, err := base64.StdEncoding.DecodeString(C.GoString(data))
	if err != nil {
		return makeJSONResponse(err)
	}

	keyInfo := &node.KeyInfo{
		Kind: node.KeyKind(C.GoString(keyType)),
	}

	keyBytes, err := hexutil.Decode(C.GoString(key))
	if err != nil {
		return makeJSONResponse(err)
	}

	switch keyInfo.Kind {
	case node.Symmetric:
		keyInfo.SymKey = keyBytes
	case node.Asymmetric:
		privK, err := crypto.ToECDSA(keyBytes)
		if err != nil {
			return makeJSONResponse(err)
		}
		keyInfo.PrivKey = privK
	}

	msg := pb.WakuMessage{
		Payload: b,
		Version: uint32(version),
	}
	payload, err := node.DecodePayload(&msg, keyInfo)
	if err != nil {
		return makeJSONResponse(err)
	}

	response := struct {
		PubKey    string `json:"pubkey"`
		Signature string `json:"signature"`
		Data      []byte `json:"data"`
		Padding   []byte `json:"padding"`
	}{
		PubKey:    hexutil.Encode(crypto.FromECDSAPub(payload.PubKey)),
		Signature: hexutil.Encode(payload.Signature),
		Data:      payload.Data,
		Padding:   payload.Padding,
	}

	return prepareJSONResponse(response, err)
}

//export gowaku_utils_base64_decode
// Decode a base64 string (useful for reading the payload from waku messages)
func gowaku_utils_base64_decode(data *C.char) *C.char {
	b, err := base64.StdEncoding.DecodeString(C.GoString(data))
	if err != nil {
		return makeJSONResponse(err)
	}

	return prepareJSONResponse(string(b), nil)
}

//export gowaku_utils_base64_encode
// Encode data to base64 (useful for creating the payload of a waku message in the
// format understood by gowaku_relay_publish)
func gowaku_utils_base64_encode(data *C.char) *C.char {
	str := base64.StdEncoding.EncodeToString([]byte(C.GoString(data)))
	return C.CString(string(str))

}

// TODO:
// connected/disconnected
// dns discovery
// func gowaku_relay_publish_msg(msg C.WakuMessage, pubsubTopic *C.char, ms C.int) *C.char
// getFastestPeer(protocol)
// getRandomPeer(protocol)
// func (wakuLP *WakuLightPush) PublishToTopic(ctx context.Context, message *pb.WakuMessage, topic string, peer, requestId nil) ([]byte, error) {
// func (wakuLP *WakuLightPush) Publish(ctx context.Context, message *pb.WakuMessage, peer, requestId nil) ([]byte, error) {
// func (query)
