package gowaku

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/status-im/go-waku/waku/v2/node"
	"github.com/status-im/go-waku/waku/v2/protocol"
	"github.com/status-im/go-waku/waku/v2/protocol/pb"
)

var wakuNode *node.WakuNode

var errWakuNodeNotReady = errors.New("go-waku not initialized")

func randomHex(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

type wakuConfig struct {
	Host              *string `json:"host,omitempty"`
	Port              *int    `json:"port,omitempty"`
	AdvertiseAddress  *string `json:"advertiseAddr,omitempty"`
	NodeKey           *string `json:"nodeKey,omitempty"`
	KeepAliveInterval *int    `json:"keepAliveInterval,omitempty"`
	EnableRelay       *bool   `json:"relay"`
	MinPeersToPublish *int    `json:"minPeersToPublish"`
}

var defaultHost = "0.0.0.0"
var defaultPort = 60000
var defaultKeepAliveInterval = 20
var defaultEnableRelay = true
var defaultMinPeersToPublish = 0

func getConfig(configJSON string) (wakuConfig, error) {
	var config wakuConfig
	if configJSON != "" {
		err := json.Unmarshal([]byte(configJSON), &config)
		if err != nil {
			return wakuConfig{}, err
		}
	}

	if config.Host == nil {
		config.Host = &defaultHost
	}

	if config.EnableRelay == nil {
		config.EnableRelay = &defaultEnableRelay
	}

	if config.Host == nil {
		config.Host = &defaultHost
	}

	if config.Port == nil {
		config.Port = &defaultPort
	}

	if config.KeepAliveInterval == nil {
		config.KeepAliveInterval = &defaultKeepAliveInterval
	}

	if config.MinPeersToPublish == nil {
		config.MinPeersToPublish = &defaultMinPeersToPublish
	}

	return config, nil
}

func NewNode(configJSON string) string {
	if wakuNode != nil {
		return makeJSONResponse(errors.New("go-waku already initialized. stop it first"))
	}

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

func Start() string {
	if wakuNode == nil {
		return makeJSONResponse(errWakuNodeNotReady)
	}

	if err := wakuNode.Start(); err != nil {
		return makeJSONResponse(err)
	}

	return makeJSONResponse(nil)
}

func Stop() string {
	if wakuNode == nil {
		return makeJSONResponse(errWakuNodeNotReady)
	}

	wakuNode.Stop()
	wakuNode = nil

	return makeJSONResponse(nil)
}

func PeerID() string {
	if wakuNode == nil {
		return makeJSONResponse(errWakuNodeNotReady)
	}

	return prepareJSONResponse(wakuNode.ID(), nil)
}

func ListenAddresses() string {
	if wakuNode == nil {
		return makeJSONResponse(errWakuNodeNotReady)
	}

	var addresses []string
	for _, addr := range wakuNode.ListenAddresses() {
		addresses = append(addresses, addr.String())
	}

	return prepareJSONResponse(addresses, nil)
}

func AddPeer(address string, protocolID string) string {
	if wakuNode == nil {
		return makeJSONResponse(errWakuNodeNotReady)
	}

	ma, err := multiaddr.NewMultiaddr(address)
	if err != nil {
		return makeJSONResponse(err)
	}

	peerID, err := wakuNode.AddPeer(ma, protocolID)
	return prepareJSONResponse(peerID, err)
}

func Connect(address string, ms int) string {
	if wakuNode == nil {
		return makeJSONResponse(errWakuNodeNotReady)
	}

	var ctx context.Context
	var cancel context.CancelFunc

	if ms > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(int(ms))*time.Millisecond)
		defer cancel()
	} else {
		ctx = context.Background()
	}

	err := wakuNode.DialPeer(ctx, address)
	return makeJSONResponse(err)
}

func ConnectPeerID(peerID string, ms int) string {
	if wakuNode == nil {
		return makeJSONResponse(errWakuNodeNotReady)
	}

	var ctx context.Context
	var cancel context.CancelFunc

	pID, err := peer.Decode(peerID)
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

func Disconnect(peerID string) string {
	if wakuNode == nil {
		return makeJSONResponse(errWakuNodeNotReady)
	}

	pID, err := peer.Decode(peerID)
	if err != nil {
		return makeJSONResponse(err)
	}

	err = wakuNode.ClosePeerById(pID)
	return makeJSONResponse(err)
}

func PeerCnt() string {
	if wakuNode == nil {
		return makeJSONResponse(errWakuNodeNotReady)
	}

	return prepareJSONResponse(wakuNode.PeerCount(), nil)
}

func ContentTopic(applicationName string, applicationVersion int, contentTopicName string, encoding string) string {
	return protocol.NewContentTopic(applicationName, uint(applicationVersion), contentTopicName, encoding).String()
}

func PubsubTopic(name string, encoding string) string {
	return protocol.NewPubsubTopic(name, encoding).String()
}

func DefaultPubsubTopic() string {
	return protocol.DefaultPubsubTopic().String()
}

func getTopic(topic string) string {
	if topic == "" {
		return protocol.DefaultPubsubTopic().String()
	}
	return topic
}

type subscriptionMsg struct {
	MessageID   string          `json:"messageID"`
	PubsubTopic string          `json:"pubsubTopic"`
	Message     *pb.WakuMessage `json:"wakuMessage"`
}

func toSubscriptionMessage(msg *protocol.Envelope) *subscriptionMsg {
	return &subscriptionMsg{
		MessageID:   hexutil.Encode(msg.Hash()),
		PubsubTopic: msg.PubsubTopic(),
		Message:     msg.Message(),
	}
}

func Peers() string {
	if wakuNode == nil {
		return makeJSONResponse(errWakuNodeNotReady)
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

func DecodeSymmetric(messageJSON string, symmetricKey string) string {
	var msg pb.WakuMessage
	err := json.Unmarshal([]byte(messageJSON), &msg)
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

	keyInfo.SymKey, err = hexutil.Decode(symmetricKey)
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

func DecodeAsymmetric(messageJSON string, privateKey string) string {
	var msg pb.WakuMessage
	err := json.Unmarshal([]byte(messageJSON), &msg)
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

	keyBytes, err := hexutil.Decode(privateKey)
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
