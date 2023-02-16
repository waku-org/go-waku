// Implements gomobile bindings for go-waku. Contains a set of functions that
// are exported when go-waku is exported as libraries for mobile devices
package gowaku

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"time"

	"go.uber.org/zap/zapcore"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/libp2p/go-libp2p/core/peer"
	libp2pProtocol "github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"
	"github.com/waku-org/go-waku/waku"
	"github.com/waku-org/go-waku/waku/persistence"
	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-waku/waku/v2/payload"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

type WakuState struct {
	ctx    context.Context
	cancel context.CancelFunc

	node *node.WakuNode

	relayTopics []string
}

var wakuState WakuState

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
	RelayTopics          []string `json:"relayTopics,omitempty"`
	EnableFilter         *bool    `json:"filter,omitempty"`
	MinPeersToPublish    *int     `json:"minPeersToPublish,omitempty"`
	EnableDiscV5         *bool    `json:"discV5,omitempty"`
	DiscV5BootstrapNodes []string `json:"discV5BootstrapNodes,omitempty"`
	DiscV5UDPPort        *uint    `json:"discV5UDPPort,omitempty"`
	EnableStore          *bool    `json:"store,omitempty"`
	DatabaseURL          *string  `json:"databaseURL,omitempty"`
	RetentionMaxMessages *int     `json:"storeRetentionMaxMessages,omitempty"`
	RetentionTimeSeconds *int     `json:"storeRetentionTimeSeconds,omitempty"`
}

var defaultHost = "0.0.0.0"
var defaultPort = 60000
var defaultKeepAliveInterval = 20
var defaultEnableRelay = true
var defaultMinPeersToPublish = 0
var defaultEnableFilter = false
var defaultEnableDiscV5 = false
var defaultDiscV5UDPPort = uint(9000)
var defaultLogLevel = "INFO"
var defaultEnableStore = false
var defaultDatabaseURL = "sqlite3://store.db"
var defaultRetentionMaxMessages = 10000
var defaultRetentionTimeSeconds = 30 * 24 * 60 * 60 // 30d

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

	if config.EnableStore == nil {
		config.EnableStore = &defaultEnableStore
	}

	if config.DatabaseURL == nil {
		config.DatabaseURL = &defaultDatabaseURL
	}

	if config.RetentionMaxMessages == nil {
		config.RetentionMaxMessages = &defaultRetentionMaxMessages
	}

	if config.RetentionTimeSeconds == nil {
		config.RetentionTimeSeconds = &defaultRetentionTimeSeconds
	}

	return config, nil
}

func NewNode(configJSON string) string {
	if wakuState.node != nil {
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
		node.NoDefaultWakuTopic(),
	}

	if *config.EnableRelay {
		opts = append(opts, node.WithWakuRelayAndMinPeers(*config.MinPeersToPublish))
	}

	if *config.EnableFilter {
		opts = append(opts, node.WithWakuFilter(false))
	}

	if *config.EnableStore {
		var db *sql.DB
		var migrationFn func(*sql.DB) error
		db, migrationFn, err = waku.ExtractDBAndMigration(*config.DatabaseURL)
		if err != nil {
			return MakeJSONResponse(err)
		}
		opts = append(opts, node.WithWakuStore())
		dbStore, err := persistence.NewDBStore(utils.Logger(),
			persistence.WithDB(db),
			persistence.WithMigrations(migrationFn),
			persistence.WithRetentionPolicy(*config.RetentionMaxMessages, time.Duration(*config.RetentionTimeSeconds)*time.Second),
		)
		if err != nil {
			return MakeJSONResponse(err)
		}
		opts = append(opts, node.WithMessageProvider(dbStore))
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
		opts = append(opts, node.WithDiscoveryV5(*config.DiscV5UDPPort, bootnodes, true))
	}

	wakuState.relayTopics = config.RelayTopics

	lvl, err := zapcore.ParseLevel(*config.LogLevel)
	if err != nil {
		return MakeJSONResponse(err)
	}

	opts = append(opts, node.WithLogLevel(lvl))

	w, err := node.New(opts...)
	if err != nil {
		return MakeJSONResponse(err)
	}

	wakuState.node = w

	return MakeJSONResponse(nil)
}

func Start() string {
	if wakuState.node == nil {
		return MakeJSONResponse(errWakuNodeNotReady)
	}

	wakuState.ctx, wakuState.cancel = context.WithCancel(context.Background())

	if err := wakuState.node.Start(wakuState.ctx); err != nil {
		return MakeJSONResponse(err)
	}

	if wakuState.node.DiscV5() != nil {
		if err := wakuState.node.DiscV5().Start(context.Background()); err != nil {
			wakuState.node.Stop()
			return MakeJSONResponse(err)
		}
	}

	for _, topic := range wakuState.relayTopics {
		topic := topic
		sub, err := wakuState.node.Relay().SubscribeToTopic(wakuState.ctx, topic)
		if err != nil {
			wakuState.node.Stop()
			return MakeJSONResponse(fmt.Errorf("could not subscribe to topic: %s, %w", topic, err))
		}
		wakuState.node.Broadcaster().Unregister(&topic, sub.C)
	}

	return MakeJSONResponse(nil)
}

func Stop() string {
	if wakuState.node == nil {
		return MakeJSONResponse(errWakuNodeNotReady)
	}

	wakuState.node.Stop()

	wakuState.cancel()

	wakuState.node = nil

	return MakeJSONResponse(nil)
}

func IsStarted() string {
	return PrepareJSONResponse(wakuState.node != nil, nil)
}

func PeerID() string {
	if wakuState.node == nil {
		return MakeJSONResponse(errWakuNodeNotReady)
	}

	return PrepareJSONResponse(wakuState.node.ID(), nil)
}

func ListenAddresses() string {
	if wakuState.node == nil {
		return MakeJSONResponse(errWakuNodeNotReady)
	}

	var addresses []string
	for _, addr := range wakuState.node.ListenAddresses() {
		addresses = append(addresses, addr.String())
	}

	return PrepareJSONResponse(addresses, nil)
}

func AddPeer(address string, protocolID libp2pProtocol.ID) string {
	if wakuState.node == nil {
		return MakeJSONResponse(errWakuNodeNotReady)
	}

	ma, err := multiaddr.NewMultiaddr(address)
	if err != nil {
		return MakeJSONResponse(err)
	}

	peerID, err := wakuState.node.AddPeer(ma, protocolID)
	return PrepareJSONResponse(peerID, err)
}

func Connect(address string, ms int) string {
	if wakuState.node == nil {
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

	err := wakuState.node.DialPeer(ctx, address)
	return MakeJSONResponse(err)
}

func ConnectPeerID(peerID string, ms int) string {
	if wakuState.node == nil {
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

	err = wakuState.node.DialPeerByID(ctx, pID)
	return MakeJSONResponse(err)
}

func Disconnect(peerID string) string {
	if wakuState.node == nil {
		return MakeJSONResponse(errWakuNodeNotReady)
	}

	pID, err := peer.Decode(peerID)
	if err != nil {
		return MakeJSONResponse(err)
	}

	err = wakuState.node.ClosePeerById(pID)
	return MakeJSONResponse(err)
}

func PeerCnt() string {
	if wakuState.node == nil {
		return MakeJSONResponse(errWakuNodeNotReady)
	}

	return PrepareJSONResponse(wakuState.node.PeerCount(), nil)
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
	if wakuState.node == nil {
		return MakeJSONResponse(errWakuNodeNotReady)
	}

	peers, err := wakuState.node.Peers()
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
