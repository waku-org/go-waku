// Implements gomobile bindings for go-waku. Contains a set of functions that
// are exported when go-waku is exported as libraries for mobile devices
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

	logging "github.com/ipfs/go-log/v2"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/ethereum/go-ethereum/p2p/enode"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/discovery"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-waku/waku/v2/payload"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

var wakuNode *node.WakuNode
var wakuStarted = false

var errWakuNodeNotReady = errors.New("go-waku not initialized")

func randomHex(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

type wakuConfig struct {
	Host                 *string  `json:"host,omitempty"`
	Port                 *int     `json:"port,omitempty"`
	AdvertiseAddress     *string  `json:"advertiseAddr,omitempty"`
	NodeKey              *string  `json:"nodeKey,omitempty"`
	LogLevel             *string  `json:"logLevel,omitempty"`
	KeepAliveInterval    *int     `json:"keepAliveInterval,omitempty"`
	EnableRelay          *bool    `json:"relay"`
	EnableFilter         *bool    `json:"filter"`
	MinPeersToPublish    *int     `json:"minPeersToPublish"`
	EnableDiscV5         *bool    `json:"discV5"`
	DiscV5BootstrapNodes []string `json:"discV5BootstrapNodes"`
	DiscV5UDPPort        *int     `json:"discV5UDPPort"`
}

var defaultHost = "0.0.0.0"
var defaultPort = 60000
var defaultKeepAliveInterval = 20
var defaultEnableRelay = true
var defaultMinPeersToPublish = 0
var defaultEnableFilter = false
var defaultEnableDiscV5 = false
var defaultDiscV5UDPPort = 9000
var defaultLogLevel = "INFO"

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

	if config.EnableFilter == nil {
		config.EnableFilter = &defaultEnableFilter
	}

	if config.EnableDiscV5 == nil {
		config.EnableDiscV5 = &defaultEnableDiscV5
	}

	if config.Host == nil {
		config.Host = &defaultHost
	}

	if config.Port == nil {
		config.Port = &defaultPort
	}

	if config.DiscV5UDPPort == nil {
		config.DiscV5UDPPort = &defaultDiscV5UDPPort
	}

	if config.KeepAliveInterval == nil {
		config.KeepAliveInterval = &defaultKeepAliveInterval
	}

	if config.MinPeersToPublish == nil {
		config.MinPeersToPublish = &defaultMinPeersToPublish
	}

	if config.LogLevel == nil {
		config.LogLevel = &defaultLogLevel
	}

	return config, nil
}

func NewNode(configJSON string) string {
	if wakuNode != nil {
		return MakeJSONResponse(errors.New("go-waku already initialized. stop it first"))
	}

	config, err := getConfig(configJSON)
	if err != nil {
		return MakeJSONResponse(err)
	}

	hostAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", *config.Host, *config.Port))
	if err != nil {
		return MakeJSONResponse(err)
	}

	var prvKey *ecdsa.PrivateKey
	if config.NodeKey != nil {
		prvKey, err = crypto.HexToECDSA(*config.NodeKey)
		if err != nil {
			return MakeJSONResponse(err)
		}
	} else {
		key, err := randomHex(32)
		if err != nil {
			return MakeJSONResponse(err)
		}
		prvKey, err = crypto.HexToECDSA(key)
		if err != nil {
			return MakeJSONResponse(err)
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

	if *config.EnableFilter {
		opts = append(opts, node.WithWakuFilter(false))
	}

	if *config.EnableDiscV5 {
		var bootnodes []*enode.Node
		for _, addr := range config.DiscV5BootstrapNodes {
			bootnode, err := enode.Parse(enode.ValidSchemes, addr)
			if err != nil {
				return MakeJSONResponse(err)
			}
			bootnodes = append(bootnodes, bootnode)
		}
		opts = append(opts, node.WithDiscoveryV5(*config.DiscV5UDPPort, bootnodes, true, pubsub.WithDiscoveryOpts(discovery.Limit(45), discovery.TTL(time.Duration(20)*time.Second))))
	}

	// for go-libp2p loggers
	lvl, err := logging.LevelFromString(*config.LogLevel)
	if err != nil {
		return MakeJSONResponse(err)
	}
	logging.SetAllLoggers(lvl)

	ctx := context.Background()
	w, err := node.New(ctx, opts...)

	if err != nil {
		return MakeJSONResponse(err)
	}

	wakuNode = w

	return MakeJSONResponse(nil)
}

func Start() string {
	if wakuNode == nil {
		return MakeJSONResponse(errWakuNodeNotReady)
	}

	if err := wakuNode.Start(); err != nil {
		return MakeJSONResponse(err)
	}

	if wakuNode.DiscV5() != nil {
		if err := wakuNode.DiscV5().Start(context.Background()); err != nil {
			wakuNode.Stop()
			return MakeJSONResponse(err)
		}
	}

	wakuStarted = true

	return MakeJSONResponse(nil)
}

func Stop() string {
	if wakuNode == nil {
		return MakeJSONResponse(errWakuNodeNotReady)
	}

	wakuNode.Stop()

	wakuStarted = false
	wakuNode = nil

	return MakeJSONResponse(nil)
}

func IsStarted() string {
	return PrepareJSONResponse(wakuStarted, nil)
}

func PeerID() string {
	if wakuNode == nil {
		return MakeJSONResponse(errWakuNodeNotReady)
	}

	return PrepareJSONResponse(wakuNode.ID(), nil)
}

func ListenAddresses() string {
	if wakuNode == nil {
		return MakeJSONResponse(errWakuNodeNotReady)
	}

	var addresses []string
	for _, addr := range wakuNode.ListenAddresses() {
		addresses = append(addresses, addr.String())
	}

	return PrepareJSONResponse(addresses, nil)
}

func AddPeer(address string, protocolID string) string {
	if wakuNode == nil {
		return MakeJSONResponse(errWakuNodeNotReady)
	}

	ma, err := multiaddr.NewMultiaddr(address)
	if err != nil {
		return MakeJSONResponse(err)
	}

	peerID, err := wakuNode.AddPeer(ma, protocolID)
	return PrepareJSONResponse(peerID, err)
}

func Connect(address string, ms int) string {
	if wakuNode == nil {
		return MakeJSONResponse(errWakuNodeNotReady)
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
	return MakeJSONResponse(err)
}

func ConnectPeerID(peerID string, ms int) string {
	if wakuNode == nil {
		return MakeJSONResponse(errWakuNodeNotReady)
	}

	var ctx context.Context
	var cancel context.CancelFunc

	pID, err := peer.Decode(peerID)
	if err != nil {
		return MakeJSONResponse(err)
	}

	if ms > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(int(ms))*time.Millisecond)
		defer cancel()
	} else {
		ctx = context.Background()
	}

	err = wakuNode.DialPeerByID(ctx, pID)
	return MakeJSONResponse(err)
}

func Disconnect(peerID string) string {
	if wakuNode == nil {
		return MakeJSONResponse(errWakuNodeNotReady)
	}

	pID, err := peer.Decode(peerID)
	if err != nil {
		return MakeJSONResponse(err)
	}

	err = wakuNode.ClosePeerById(pID)
	return MakeJSONResponse(err)
}

func PeerCnt() string {
	if wakuNode == nil {
		return MakeJSONResponse(errWakuNodeNotReady)
	}

	return PrepareJSONResponse(wakuNode.PeerCount(), nil)
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
	MessageID   string          `json:"messageId"`
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
		return MakeJSONResponse(errWakuNodeNotReady)
	}

	peers, err := wakuNode.Peers()
	return PrepareJSONResponse(peers, err)
}

func unmarshalPubkey(pub []byte) (ecdsa.PublicKey, error) {
	x, y := elliptic.Unmarshal(secp256k1.S256(), pub)
	if x == nil {
		return ecdsa.PublicKey{}, errors.New("invalid public key")
	}
	return ecdsa.PublicKey{Curve: secp256k1.S256(), X: x, Y: y}, nil
}

func extractPubKeyAndSignature(payload *payload.DecodedPayload) (pubkey string, signature string) {
	pkBytes := crypto.FromECDSAPub(payload.PubKey)
	if len(pkBytes) != 0 {
		pubkey = hexutil.Encode(pkBytes)
	}

	if len(payload.Signature) != 0 {
		signature = hexutil.Encode(payload.Signature)
	}

	return
}

func DecodeSymmetric(messageJSON string, symmetricKey string) string {
	var msg pb.WakuMessage
	err := json.Unmarshal([]byte(messageJSON), &msg)
	if err != nil {
		return MakeJSONResponse(err)
	}

	if msg.Version == 0 {
		return PrepareJSONResponse(msg.Payload, nil)
	} else if msg.Version > 1 {
		return MakeJSONResponse(errors.New("unsupported wakumessage version"))
	}

	keyInfo := &payload.KeyInfo{
		Kind: payload.Symmetric,
	}

	keyInfo.SymKey, err = utils.DecodeHexString(symmetricKey)
	if err != nil {
		return MakeJSONResponse(err)
	}

	payload, err := payload.DecodePayload(&msg, keyInfo)
	if err != nil {
		return MakeJSONResponse(err)
	}

	pubkey, signature := extractPubKeyAndSignature(payload)

	response := struct {
		PubKey    string `json:"pubkey,omitempty"`
		Signature string `json:"signature,omitempty"`
		Data      []byte `json:"data"`
		Padding   []byte `json:"padding"`
	}{
		PubKey:    pubkey,
		Signature: signature,
		Data:      payload.Data,
		Padding:   payload.Padding,
	}

	return PrepareJSONResponse(response, err)
}

func DecodeAsymmetric(messageJSON string, privateKey string) string {
	var msg pb.WakuMessage
	err := json.Unmarshal([]byte(messageJSON), &msg)
	if err != nil {
		return MakeJSONResponse(err)
	}

	if msg.Version == 0 {
		return PrepareJSONResponse(msg.Payload, nil)
	} else if msg.Version > 1 {
		return MakeJSONResponse(errors.New("unsupported wakumessage version"))
	}

	keyInfo := &payload.KeyInfo{
		Kind: payload.Asymmetric,
	}

	keyBytes, err := utils.DecodeHexString(privateKey)
	if err != nil {
		return MakeJSONResponse(err)
	}

	keyInfo.PrivKey, err = crypto.ToECDSA(keyBytes)
	if err != nil {
		return MakeJSONResponse(err)
	}

	payload, err := payload.DecodePayload(&msg, keyInfo)
	if err != nil {
		return MakeJSONResponse(err)
	}

	pubkey, signature := extractPubKeyAndSignature(payload)

	response := struct {
		PubKey    string `json:"pubkey,omitempty"`
		Signature string `json:"signature,omitempty"`
		Data      []byte `json:"data"`
		Padding   []byte `json:"padding"`
	}{
		PubKey:    pubkey,
		Signature: signature,
		Data:      payload.Data,
		Padding:   payload.Padding,
	}

	return PrepareJSONResponse(response, err)
}
