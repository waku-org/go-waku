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
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
	libp2pProtocol "github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"
	"github.com/waku-org/go-waku/waku/persistence"
	dbutils "github.com/waku-org/go-waku/waku/persistence/utils"
	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-waku/waku/v2/payload"
	"github.com/waku-org/go-waku/waku/v2/peers"
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
	}

	if *config.EnableRelay {
		var pubsubOpt []pubsub.Option
		if config.GossipSubParams != nil {
			params := GetGossipSubParams(config.GossipSubParams)
			pubsubOpt = append(pubsubOpt, pubsub.WithGossipSubParams(params))
		}

		pubsubOpt = append(pubsubOpt, pubsub.WithSeenMessagesTTL(getSeenTTL(config)))

		opts = append(opts, node.WithWakuRelayAndMinPeers(*config.MinPeersToPublish, pubsubOpt...))
	}

	if config.DNS4DomainName != "" {
		opts = append(opts, node.WithDns4Domain(config.DNS4DomainName))
	}

	if config.Websockets.Enabled {
		if config.Websockets.Secure {
			if config.DNS4DomainName == "" {
				utils.Logger().Warn("using secure websockets without a dns4 domain name might indicate a misconfiguration")
			}
			opts = append(opts, node.WithSecureWebsockets(config.Websockets.Host, *config.Websockets.Port, config.Websockets.CertPath, config.Websockets.KeyPath))
		} else {
			opts = append(opts, node.WithWebsockets(config.Websockets.Host, *config.Websockets.Port))

		}
	}

	if *config.EnableLegacyFilter {
		opts = append(opts, node.WithLegacyWakuFilter(false))
	}

	opts = append(opts, node.WithWakuFilterLightNode())

	if *config.EnableStore {
		var db *sql.DB
		var migrationFn func(*sql.DB) error
		db, migrationFn, err = dbutils.ExtractDBAndMigration(*config.DatabaseURL)
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
		err := relaySubscribe(topic)
		if err != nil {
			wakuState.node.Stop()
			return MakeJSONResponse(err)
		}
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

func AddPeer(address string, protocolID string) string {
	if wakuState.node == nil {
		return MakeJSONResponse(errWakuNodeNotReady)
	}

	ma, err := multiaddr.NewMultiaddr(address)
	if err != nil {
		return MakeJSONResponse(err)
	}

	peerID, err := wakuState.node.AddPeer(ma, peers.Static, libp2pProtocol.ID(protocolID))
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
	return protocol.NewNamedShardingPubsubTopic(name + "/" + encoding).String()
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
