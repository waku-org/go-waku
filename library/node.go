package library

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"time"

	"go.uber.org/zap/zapcore"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enode"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
	libp2pProtocol "github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/waku-org/go-waku/waku/persistence"
	dbutils "github.com/waku-org/go-waku/waku/persistence/utils"
	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-waku/waku/v2/peerstore"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

// WakuState represents the state of the waku node
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

// NewNode initializes a waku node. Receives a JSON string containing the configuration, and use default values for those config items not specified
func NewNode(configJSON string) error {
	if wakuState.node != nil {
		return errors.New("go-waku already initialized. stop it first")
	}

	config, err := getConfig(configJSON)
	if err != nil {
		return err
	}

	hostAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", *config.Host, *config.Port))
	if err != nil {
		return err
	}

	var prvKey *ecdsa.PrivateKey
	if config.NodeKey != nil {
		prvKey, err = crypto.HexToECDSA(*config.NodeKey)
		if err != nil {
			return err
		}
	} else {
		key, err := randomHex(32)
		if err != nil {
			return err
		}
		prvKey, err = crypto.HexToECDSA(key)
		if err != nil {
			return err
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
			params := getGossipSubParams(config.GossipSubParams)
			pubsubOpt = append(pubsubOpt, pubsub.WithGossipSubParams(params))
		}

		pubsubOpt = append(pubsubOpt, pubsub.WithSeenMessagesTTL(getSeenTTL(config)))

		opts = append(opts, node.WithWakuRelayAndMinPeers(*config.MinPeersToPublish, pubsubOpt...))
	}

	if config.DNS4DomainName != "" {
		opts = append(opts, node.WithDNS4Domain(config.DNS4DomainName))
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
		db, migrationFn, err = dbutils.ExtractDBAndMigration(*config.DatabaseURL, dbutils.DBSettings{Vacuum: true}, utils.Logger())
		if err != nil {
			return err
		}
		opts = append(opts, node.WithWakuStore())
		dbStore, err := persistence.NewDBStore(prometheus.DefaultRegisterer, utils.Logger(),
			persistence.WithDB(db),
			persistence.WithMigrations(migrationFn),
			persistence.WithRetentionPolicy(*config.RetentionMaxMessages, time.Duration(*config.RetentionTimeSeconds)*time.Second),
		)
		if err != nil {
			return err
		}
		opts = append(opts, node.WithMessageProvider(dbStore))
	}

	if *config.EnableDiscV5 {
		var bootnodes []*enode.Node
		for _, addr := range config.DiscV5BootstrapNodes {
			bootnode, err := enode.Parse(enode.ValidSchemes, addr)
			if err != nil {
				return err
			}
			bootnodes = append(bootnodes, bootnode)
		}
		opts = append(opts, node.WithDiscoveryV5(*config.DiscV5UDPPort, bootnodes, true))
	}

	wakuState.relayTopics = config.RelayTopics

	lvl, err := zapcore.ParseLevel(*config.LogLevel)
	if err != nil {
		return err
	}

	opts = append(opts, node.WithLogLevel(lvl))
	opts = append(opts, node.WithPrometheusRegisterer(prometheus.DefaultRegisterer))

	w, err := node.New(opts...)
	if err != nil {
		return err
	}

	wakuState.node = w

	return nil
}

// Start starts the waku node
func Start() error {
	if wakuState.node == nil {
		return errWakuNodeNotReady
	}

	wakuState.ctx, wakuState.cancel = context.WithCancel(context.Background())

	if err := wakuState.node.Start(wakuState.ctx); err != nil {
		return err
	}

	if wakuState.node.DiscV5() != nil {
		if err := wakuState.node.DiscV5().Start(context.Background()); err != nil {
			wakuState.node.Stop()
			return err
		}
	}

	for _, topic := range wakuState.relayTopics {
		err := relaySubscribe(topic)
		if err != nil {
			wakuState.node.Stop()
			return err
		}
	}

	return nil
}

// Stop stops a waku node
func Stop() error {
	if wakuState.node == nil {
		return errWakuNodeNotReady
	}

	wakuState.node.Stop()

	wakuState.cancel()

	wakuState.node = nil

	return nil
}

// IsStarted is used to determine is a node is started or not
func IsStarted() bool {
	return wakuState.node != nil
}

// PeerID is used to obtain the peer ID of the waku node
func PeerID() (string, error) {
	if wakuState.node == nil {
		return "", errWakuNodeNotReady
	}

	return wakuState.node.ID(), nil
}

// ListenAddresses returns the multiaddresses the wakunode is listening to
func ListenAddresses() ([]string, error) {
	if wakuState.node == nil {
		return nil, errWakuNodeNotReady
	}

	var addresses []string
	for _, addr := range wakuState.node.ListenAddresses() {
		addresses = append(addresses, addr.String())
	}

	return addresses, nil
}

// AddPeer adds a node multiaddress and protocol to the wakunode peerstore
func AddPeer(address string, protocolID string) (string, error) {
	if wakuState.node == nil {
		return "", errWakuNodeNotReady
	}

	ma, err := multiaddr.NewMultiaddr(address)
	if err != nil {
		return "", err
	}

	peerID, err := wakuState.node.AddPeer(ma, peerstore.Static, wakuState.relayTopics, libp2pProtocol.ID(protocolID))
	if err != nil {
		return "", err
	}

	return peerID.Pretty(), nil
}

// Connect is used to connect to a peer at multiaddress. if ms > 0, cancel the function execution if it takes longer than N milliseconds
func Connect(address string, ms int) error {
	if wakuState.node == nil {
		return errWakuNodeNotReady
	}

	var ctx context.Context
	var cancel context.CancelFunc

	if ms > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(int(ms))*time.Millisecond)
		defer cancel()
	} else {
		ctx = context.Background()
	}

	return wakuState.node.DialPeer(ctx, address)
}

// ConnectPeerID is usedd to connect to a known peer by peerID. if ms > 0, cancel the function execution if it takes longer than N milliseconds
func ConnectPeerID(peerID string, ms int) error {
	if wakuState.node == nil {
		return errWakuNodeNotReady
	}

	var ctx context.Context
	var cancel context.CancelFunc

	pID, err := peer.Decode(peerID)
	if err != nil {
		return err
	}

	if ms > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(int(ms))*time.Millisecond)
		defer cancel()
	} else {
		ctx = context.Background()
	}

	return wakuState.node.DialPeerByID(ctx, pID)
}

// Disconnect closes a connection to a known peer by peerID
func Disconnect(peerID string) error {
	if wakuState.node == nil {
		return errWakuNodeNotReady
	}

	pID, err := peer.Decode(peerID)
	if err != nil {
		return err
	}

	return wakuState.node.ClosePeerById(pID)
}

// PeerCnt returns the number of connected peers
func PeerCnt() (int, error) {
	if wakuState.node == nil {
		return 0, errWakuNodeNotReady
	}

	return wakuState.node.PeerCount(), nil
}

// ContentTopic creates a content topic string according to RFC 23
func ContentTopic(applicationName string, applicationVersion int, contentTopicName string, encoding string) string {
	contentTopic, _ := protocol.NewContentTopic(applicationName, uint32(applicationVersion), contentTopicName, encoding)
	return contentTopic.String()
}

// PubsubTopic creates a pubsub topic string according to RFC 23
func PubsubTopic(name string, encoding string) string {
	return protocol.NewNamedShardingPubsubTopic(name + "/" + encoding).String()
}

// DefaultPubsubTopic returns the default pubsub topic used in waku2: /waku/2/default-waku/proto
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

// Peers retrieves the list of peers known by the waku node
func Peers() ([]*node.Peer, error) {
	if wakuState.node == nil {
		return nil, errWakuNodeNotReady
	}

	peers, err := wakuState.node.Peers()
	if err != nil {
		return nil, err
	}

	for _, p := range peers {
		addrs := []multiaddr.Multiaddr{}
		for i := range p.Addrs {
			_, err := p.Addrs[i].ValueForProtocol(multiaddr.P_SNI)
			if err != nil {
				addrs = append(addrs, p.Addrs[i])
			}
		}
		p.Addrs = addrs
	}

	return peers, nil
}
