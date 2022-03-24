package main

/*
#include <stdlib.h>
#include <stddef.h>
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

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/libp2p/go-libp2p-core/peer"
	p2pproto "github.com/libp2p/go-libp2p-core/protocol"
	"github.com/multiformats/go-multiaddr"
	"github.com/status-im/go-waku/waku/v2/node"
	"github.com/status-im/go-waku/waku/v2/protocol"
	"github.com/status-im/go-waku/waku/v2/protocol/pb"
)

var wakuNode *node.WakuNode

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

//export waku_new
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
func waku_new(configJSON *C.char) *C.char {
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
	w, err := node.New(ctx, opts...)

	if err != nil {
		return makeJSONResponse(err)
	}

	wakuNode = w

	return makeJSONResponse(nil)
}

//export waku_start
// Starts the waku node
func waku_start() *C.char {
	if wakuNode == nil {
		return makeJSONResponse(ErrWakuNodeNotReady)
	}

	if err := wakuNode.Start(); err != nil {
		return makeJSONResponse(err)
	}

	return makeJSONResponse(nil)
}

//export waku_stop
// Stops a waku node
func waku_stop() *C.char {
	if wakuNode == nil {
		return makeJSONResponse(ErrWakuNodeNotReady)
	}

	wakuNode.Stop()
	wakuNode = nil

	return makeJSONResponse(nil)
}

//export waku_peerid
// Obtain the peer ID of the waku node
func waku_peerid() *C.char {
	if wakuNode == nil {
		return makeJSONResponse(ErrWakuNodeNotReady)
	}

	return prepareJSONResponse(wakuNode.ID(), nil)
}

//export waku_listen_addresses
// Obtain the multiaddresses the wakunode is listening to
func waku_listen_addresses() *C.char {
	if wakuNode == nil {
		return makeJSONResponse(ErrWakuNodeNotReady)
	}

	var addresses []string
	for _, addr := range wakuNode.ListenAddresses() {
		addresses = append(addresses, addr.String())
	}

	return prepareJSONResponse(addresses, nil)
}

//export waku_add_peer
// Add node multiaddress and protocol to the wakunode peerstore
func waku_add_peer(address *C.char, protocolID *C.char) *C.char {
	if wakuNode == nil {
		return makeJSONResponse(ErrWakuNodeNotReady)
	}

	ma, err := multiaddr.NewMultiaddr(C.GoString(address))
	if err != nil {
		return makeJSONResponse(err)
	}

	peerID, err := wakuNode.AddPeer(ma, p2pproto.ID(C.GoString(protocolID)))
	return prepareJSONResponse(peerID, err)
}

//export waku_connect
// Connect to peer at multiaddress. if ms > 0, cancel the function execution if it takes longer than N milliseconds
func waku_connect(address *C.char, ms C.int) *C.char {
	if wakuNode == nil {
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

//export waku_connect_peerid
// Connect to known peer by peerID. if ms > 0, cancel the function execution if it takes longer than N milliseconds
func waku_connect_peerid(peerID *C.char, ms C.int) *C.char {
	if wakuNode == nil {
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

//export waku_disconnect
// Close connection to a known peer by peerID
func waku_disconnect(peerID *C.char) *C.char {
	if wakuNode == nil {
		return makeJSONResponse(ErrWakuNodeNotReady)
	}

	pID, err := peer.Decode(C.GoString(peerID))
	if err != nil {
		return makeJSONResponse(err)
	}

	err = wakuNode.ClosePeerById(pID)
	return makeJSONResponse(err)
}

//export waku_peer_cnt
// Get number of connected peers
func waku_peer_cnt() *C.char {
	if wakuNode == nil {
		return makeJSONResponse(ErrWakuNodeNotReady)
	}

	return prepareJSONResponse(wakuNode.PeerCount(), nil)
}

//export waku_content_topic
// Create a content topic string according to RFC 23
func waku_content_topic(applicationName *C.char, applicationVersion C.uint, contentTopicName *C.char, encoding *C.char) *C.char {
	return C.CString(protocol.NewContentTopic(C.GoString(applicationName), uint(applicationVersion), C.GoString(contentTopicName), C.GoString(encoding)).String())
}

//export waku_pubsub_topic
// Create a pubsub topic string according to RFC 23
func waku_pubsub_topic(name *C.char, encoding *C.char) *C.char {
	return prepareJSONResponse(protocol.NewPubsubTopic(C.GoString(name), C.GoString(encoding)).String(), nil)
}

//export waku_default_pubsub_topic
// Get the default pubsub topic used in waku2: /waku/2/default-waku/proto
func waku_default_pubsub_topic() *C.char {
	return C.CString(protocol.DefaultPubsubTopic().String())
}

func getTopic(topic *C.char) string {
	result := ""
	if topic != nil {
		result = C.GoString(topic)
	} else {
		result = protocol.DefaultPubsubTopic().String()
	}
	return result
}

//export waku_set_event_callback
// Register callback to act as signal handler and receive application signal
// (in JSON) which are used o react to asyncronous events in waku. The function
// signature for the callback should be `void myCallback(char* signalJSON)`
func waku_set_event_callback(cb unsafe.Pointer) {
	setEventCallback(cb)
}

type SubscriptionMsg struct {
	MessageID   string          `json:"messageID"`
	PubsubTopic string          `json:"pubsubTopic"`
	Message     *pb.WakuMessage `json:"wakuMessage"`
}

func toSubscriptionMessage(msg *protocol.Envelope) *SubscriptionMsg {
	return &SubscriptionMsg{
		MessageID:   hexutil.Encode(msg.Hash()),
		PubsubTopic: msg.PubsubTopic(),
		Message:     msg.Message(),
	}
}

//export waku_peers
// Retrieve the list of peers known by the waku node
func waku_peers() *C.char {
	if wakuNode == nil {
		return makeJSONResponse(ErrWakuNodeNotReady)
	}

	peers, err := wakuNode.Peers()
	return prepareJSONResponse(peers, err)
}

func unmarshalPubkey(pub []byte) (ecdsa.PublicKey, error) {
	x, y := elliptic.Unmarshal(secp256k1.S256(), pub)
	if x == nil {
		return ecdsa.PublicKey{}, errors.New("invalid public key")
	}
	return ecdsa.PublicKey{Curve: secp256k1.S256(), X: x, Y: y}, nil
}

//export waku_decode_symmetric
// Decode a waku message using a 32 bytes symmetric key. The key must be a hex encoded string with "0x" prefix
func waku_decode_symmetric(messageJSON *C.char, symmetricKey *C.char) *C.char {
	var msg pb.WakuMessage
	err := json.Unmarshal([]byte(C.GoString(messageJSON)), &msg)
	if err != nil {
		return makeJSONResponse(err)
	}

	if msg.Version == 0 {
		return prepareJSONResponse(msg.Payload, nil)
	} else if msg.Version > 1 {
		return makeJSONResponse(errors.New("unsupported wakumessage version"))
	}

	keyInfo := &node.KeyInfo{
		Kind: node.Symmetric,
	}

	keyInfo.SymKey, err = hexutil.Decode(C.GoString(symmetricKey))
	if err != nil {
		return makeJSONResponse(err)
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

//export waku_decode_asymmetric
// Decode a waku message using a secp256k1 private key. The key must be a hex encoded string with "0x" prefix
func waku_decode_asymmetric(messageJSON *C.char, privateKey *C.char) *C.char {
	var msg pb.WakuMessage
	err := json.Unmarshal([]byte(C.GoString(messageJSON)), &msg)
	if err != nil {
		return makeJSONResponse(err)
	}

	if msg.Version == 0 {
		return prepareJSONResponse(msg.Payload, nil)
	} else if msg.Version > 1 {
		return makeJSONResponse(errors.New("unsupported wakumessage version"))
	}

	keyInfo := &node.KeyInfo{
		Kind: node.Asymmetric,
	}

	keyBytes, err := hexutil.Decode(C.GoString(privateKey))
	if err != nil {
		return makeJSONResponse(err)
	}

	keyInfo.PrivKey, err = crypto.ToECDSA(keyBytes)
	if err != nil {
		return makeJSONResponse(err)
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

//export waku_utils_base64_decode
// Decode a base64 string (useful for reading the payload from waku messages)
func waku_utils_base64_decode(data *C.char) *C.char {
	b, err := base64.StdEncoding.DecodeString(C.GoString(data))
	if err != nil {
		return makeJSONResponse(err)
	}

	return prepareJSONResponse(string(b), nil)
}

//export waku_utils_base64_encode
// Encode data to base64 (useful for creating the payload of a waku message in the
// format understood by waku_relay_publish)
func waku_utils_base64_encode(data *C.char) *C.char {
	str := base64.StdEncoding.EncodeToString([]byte(C.GoString(data)))
	return C.CString(string(str))

}

//export waku_utils_free
// Frees a char* since all strings returned by gowaku are allocated in the C heap using malloc.
func waku_utils_free(data *C.char) {
	C.free(unsafe.Pointer(data))
}

// TODO:
// connected/disconnected
// dns discovery
// getFastestPeer(protocol)
// getRandomPeer(protocol)
// func (wakuLP *WakuLightPush) PublishToTopic(ctx context.Context, message *pb.WakuMessage, topic string, peer, requestId nil) ([]byte, error) {
// func (wakuLP *WakuLightPush) Publish(ctx context.Context, message *pb.WakuMessage, peer, requestId nil) ([]byte, error) {
// func (query)
