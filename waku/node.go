package waku

import (
	"context"
	"crypto/ecdsa"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	wmetrics "github.com/waku-org/go-waku/waku/v2/metrics"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enode"
	dssql "github.com/ipfs/go-ds-sql"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/config"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/p2p/host/peerstore/pstoreds"
	ws "github.com/libp2p/go-libp2p/p2p/transport/websocket"
	"github.com/multiformats/go-multiaddr"
	"github.com/waku-org/go-waku/logging"
	"github.com/waku-org/go-waku/waku/metrics"
	"github.com/waku-org/go-waku/waku/persistence"
	"github.com/waku-org/go-waku/waku/persistence/sqlite"
	"github.com/waku-org/go-waku/waku/v2/dnsdisc"
	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-waku/waku/v2/protocol/filter"
	"github.com/waku-org/go-waku/waku/v2/protocol/lightpush"
	"github.com/waku-org/go-waku/waku/v2/protocol/peer_exchange"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/protocol/store"
	"github.com/waku-org/go-waku/waku/v2/rest"
	"github.com/waku-org/go-waku/waku/v2/rpc"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

func failOnErr(err error, msg string) {
	if err != nil {
		utils.Logger().Fatal(msg, zap.Error(err))
	}
}

func freePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}

	port := l.Addr().(*net.TCPAddr).Port
	err = l.Close()
	if err != nil {
		return 0, err
	}

	return port, nil
}

const dialTimeout = 7 * time.Second

// Execute starts a go-waku node with settings determined by the Options parameter
func Execute(options Options) {
	if options.GenerateKey {
		if err := writePrivateKeyToFile(options.KeyFile, []byte(options.KeyPasswd), options.Overwrite); err != nil {
			failOnErr(err, "nodekey error")
		}
		return
	}

	hostAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", options.Address, options.Port))
	failOnErr(err, "invalid host address")

	prvKey, err := getPrivKey(options)
	failOnErr(err, "nodekey error")

	p2pPrvKey := utils.EcdsaPrivKeyToSecp256k1PrivKey(prvKey)
	id, err := peer.IDFromPublicKey(p2pPrvKey.GetPublic())
	failOnErr(err, "deriving peer ID from private key")
	logger := utils.Logger().With(logging.HostID("node", id))

	var db *sql.DB
	var migrationFn func(*sql.DB) error
	if options.Store.Enable {
		db, migrationFn, err = extractDBAndMigration(options.Store.DatabaseURL)
		failOnErr(err, "Could not connect to DB")
	}

	ctx := context.Background()

	var metricsServer *metrics.Server
	if options.Metrics.Enable {
		metricsServer = metrics.NewMetricsServer(options.Metrics.Address, options.Metrics.Port, logger)
		go metricsServer.Start()
		wmetrics.RecordVersion(ctx, node.Version, node.GitCommit)
	}

	lvl, err := zapcore.ParseLevel(options.LogLevel)
	if err != nil {
		failOnErr(err, "log level error")
	}

	nodeOpts := []node.WakuNodeOption{
		node.WithLogger(logger),
		node.WithLogLevel(lvl),
		node.WithPrivateKey(prvKey),
		node.WithHostAddress(hostAddr),
		node.WithKeepAlive(options.KeepAlive),
	}

	if options.AdvertiseAddress != "" {
		advertiseAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", options.AdvertiseAddress, options.Port))
		failOnErr(err, "Invalid advertise address")

		if advertiseAddr.Port == 0 {
			for {
				p, err := freePort()
				if err == nil {
					advertiseAddr.Port = p
					break
				}
			}
		}

		nodeOpts = append(nodeOpts, node.WithAdvertiseAddress(advertiseAddr))
	}

	if options.Dns4DomainName != "" {
		nodeOpts = append(nodeOpts, node.WithDns4Domain(options.Dns4DomainName))
	}

	libp2pOpts := node.DefaultLibP2POptions
	if options.AdvertiseAddress == "" {
		libp2pOpts = append(libp2pOpts, libp2p.NATPortMap()) // Attempt to open ports using uPNP for NATed hosts.)
	}

	if options.UserAgent != "" {
		libp2pOpts = append(libp2pOpts, libp2p.UserAgent(options.UserAgent))
	}

	if options.Websocket.Enable {
		nodeOpts = append(nodeOpts, node.WithWebsockets(options.Websocket.Address, options.Websocket.WSPort))
	}

	if options.Websocket.Secure {
		nodeOpts = append(nodeOpts, node.WithSecureWebsockets(options.Websocket.Address, options.Websocket.WSSPort, options.Websocket.CertPath, options.Websocket.KeyPath))
	}

	if options.ShowAddresses {
		printListeningAddresses(ctx, nodeOpts, options)
		return
	}

	if options.Store.Enable && options.PersistPeers {
		// Create persistent peerstore
		queries, err := sqlite.NewQueries("peerstore", db)
		failOnErr(err, "Peerstore")

		datastore := dssql.NewDatastore(db, queries)
		opts := pstoreds.DefaultOpts()
		peerStore, err := pstoreds.NewPeerstore(ctx, datastore, opts)
		failOnErr(err, "Peerstore")

		libp2pOpts = append(libp2pOpts, libp2p.Peerstore(peerStore))
	}

	nodeOpts = append(nodeOpts, node.WithLibP2POptions(libp2pOpts...))
	nodeOpts = append(nodeOpts, node.WithNTP())

	if options.Relay.Enable {
		var wakurelayopts []pubsub.Option
		wakurelayopts = append(wakurelayopts, pubsub.WithPeerExchange(options.Relay.PeerExchange))
		nodeOpts = append(nodeOpts, node.WithWakuRelayAndMinPeers(options.Relay.MinRelayPeersToPublish, wakurelayopts...))
	}

	if options.Filter.Enable {
		if options.Filter.UseV2 {
			if !options.Filter.DisableFullNode {
				nodeOpts = append(nodeOpts, node.WithWakuFilterV2LightNode())
			} else {
				nodeOpts = append(nodeOpts, node.WithWakuFilterV2FullNode(filter.WithTimeout(options.Filter.Timeout)))
			}
		} else {
			nodeOpts = append(nodeOpts, node.WithWakuFilter(!options.Filter.DisableFullNode, filter.WithTimeout(options.Filter.Timeout)))
		}
	}

	if options.Store.Enable {
		nodeOpts = append(nodeOpts, node.WithWakuStore(options.Store.ResumeNodes...))
		dbStore, err := persistence.NewDBStore(logger,
			persistence.WithDB(db),
			persistence.WithMigrations(migrationFn),
			persistence.WithRetentionPolicy(options.Store.RetentionMaxMessages, options.Store.RetentionTime),
		)
		failOnErr(err, "DBStore")
		nodeOpts = append(nodeOpts, node.WithMessageProvider(dbStore))
	}

	if options.LightPush.Enable {
		nodeOpts = append(nodeOpts, node.WithLightPush())
	}

	var discoveredNodes []dnsdisc.DiscoveredNode
	if options.DNSDiscovery.Enable {
		if len(options.DNSDiscovery.URLs.Value()) != 0 {
			for _, url := range options.DNSDiscovery.URLs.Value() {
				logger.Info("attempting DNS discovery with ", zap.String("URL", url))
				nodes, err := dnsdisc.RetrieveNodes(ctx, url, dnsdisc.WithNameserver(options.DNSDiscovery.Nameserver))
				if err != nil {
					logger.Warn("dns discovery error ", zap.Error(err))
				} else {
					var discAddresses []multiaddr.Multiaddr
					for _, n := range nodes {
						discAddresses = append(discAddresses, n.Addresses...)
					}
					logger.Info("found dns entries ", logging.MultiAddrs("nodes", discAddresses...))
					discoveredNodes = nodes
				}
			}
		} else {
			logger.Fatal("DNS discovery URL is required")
		}
	}

	if options.DiscV5.Enable {
		var bootnodes []*enode.Node
		for _, addr := range options.DiscV5.Nodes.Value() {
			bootnode, err := enode.Parse(enode.ValidSchemes, addr)
			if err != nil {
				logger.Fatal("parsing ENR", zap.Error(err))
			}
			bootnodes = append(bootnodes, bootnode)
		}

		for _, n := range discoveredNodes {
			if n.ENR != nil {
				bootnodes = append(bootnodes, n.ENR)
			}
		}

		nodeOpts = append(nodeOpts, node.WithDiscoveryV5(options.DiscV5.Port, bootnodes, options.DiscV5.AutoUpdate))
	}

	if options.PeerExchange.Enable {
		nodeOpts = append(nodeOpts, node.WithPeerExchange())
	}

	checkForRLN(logger, options, &nodeOpts)

	wakuNode, err := node.New(nodeOpts...)

	utils.Logger().Info("Version details ", zap.String("version", node.Version), zap.String("commit", node.GitCommit))

	failOnErr(err, "Wakunode")

	addPeers(wakuNode, options.Store.Nodes, string(store.StoreID_v20beta4))
	addPeers(wakuNode, options.LightPush.Nodes, string(lightpush.LightPushID_v20beta1))

	if options.Filter.UseV2 {
		addPeers(wakuNode, options.Filter.Nodes, string(filter.FilterID_v20beta1))
	} else {
		addPeers(wakuNode, options.Filter.Nodes, string(filter.FilterID_v20beta1))
	}

	if err = wakuNode.Start(ctx); err != nil {
		logger.Fatal("starting waku node", zap.Error(err))
	}

	if options.DiscV5.Enable {
		if err = wakuNode.DiscV5().Start(ctx); err != nil {
			logger.Fatal("starting discovery v5", zap.Error(err))
		}
	}

	// retrieve and connect to peer exchange peers
	if options.PeerExchange.Enable && options.PeerExchange.Node != nil {
		logger.Info("retrieving peer info via peer exchange protocol")

		peerId, err := wakuNode.AddPeer(*options.PeerExchange.Node, string(peer_exchange.PeerExchangeID_v20alpha1))
		if err != nil {
			logger.Error("adding peer exchange peer", logging.MultiAddrs("node", *options.PeerExchange.Node), zap.Error(err))
		} else {
			desiredOutDegree := 6 // TODO: obtain this from gossipsub D
			if err = wakuNode.PeerExchange().Request(ctx, desiredOutDegree, peer_exchange.WithPeer(peerId)); err != nil {
				logger.Error("requesting peers via peer exchange", zap.Error(err))
			}
		}
	}

	if len(options.Relay.Topics.Value()) == 0 {
		options.Relay.Topics = *cli.NewStringSlice(relay.DefaultWakuTopic)
	}

	if options.Relay.Enable {
		for _, nodeTopic := range options.Relay.Topics.Value() {
			nodeTopic := nodeTopic
			sub, err := wakuNode.Relay().SubscribeToTopic(ctx, nodeTopic)
			failOnErr(err, "Error subscring to topic")
			wakuNode.Broadcaster().Unregister(&nodeTopic, sub.C)
		}
	}

	for _, n := range options.StaticNodes {
		go func(ctx context.Context, node multiaddr.Multiaddr) {
			ctx, cancel := context.WithTimeout(ctx, dialTimeout)
			defer cancel()
			err = wakuNode.DialPeerWithMultiAddress(ctx, node)
			if err != nil {
				logger.Error("dialing peer", zap.Error(err))
			}
		}(ctx, n)
	}

	if len(discoveredNodes) != 0 {
		for _, n := range discoveredNodes {
			for _, m := range n.Addresses {
				go func(ctx context.Context, m multiaddr.Multiaddr) {
					ctx, cancel := context.WithTimeout(ctx, dialTimeout)
					defer cancel()
					err = wakuNode.DialPeerWithMultiAddress(ctx, m)
					if err != nil {
						logger.Error("dialing peer", logging.MultiAddrs("peer", m), zap.Error(err))
					}
				}(ctx, m)
			}
		}
	}

	onStartRLN(wakuNode, options)

	var rpcServer *rpc.WakuRpc
	if options.RPCServer.Enable {
		rpcServer = rpc.NewWakuRpc(wakuNode, options.RPCServer.Address, options.RPCServer.Port, options.RPCServer.Admin, options.RPCServer.Private, options.PProf, options.RPCServer.RelayCacheCapacity, logger)
		rpcServer.Start()
	}

	var wg sync.WaitGroup

	var restServer *rest.WakuRest
	if options.RESTServer.Enable {
		wg.Add(1)
		restServer = rest.NewWakuRest(wakuNode, options.RESTServer.Address, options.RESTServer.Port, options.RESTServer.Admin, options.RESTServer.Private, options.PProf, options.RESTServer.RelayCacheCapacity, logger)
		restServer.Start(ctx, &wg)
	}

	wg.Wait()
	logger.Info("Node setup complete")

	// Wait for a SIGINT or SIGTERM signal
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	logger.Info("Received signal, shutting down...")

	// shut the node down
	wakuNode.Stop()

	if options.RPCServer.Enable {
		err := rpcServer.Stop(ctx)
		failOnErr(err, "RPCClose")
	}

	if options.RESTServer.Enable {
		err := restServer.Stop(ctx)
		failOnErr(err, "RESTClose")
	}

	if options.Metrics.Enable {
		err = metricsServer.Stop(ctx)
		failOnErr(err, "MetricsClose")
	}

	if options.Store.Enable {
		err = db.Close()
		failOnErr(err, "DBClose")
	}
}

func addPeers(wakuNode *node.WakuNode, addresses []multiaddr.Multiaddr, protocols ...string) {
	for _, addr := range addresses {
		_, err := wakuNode.AddPeer(addr, protocols...)
		failOnErr(err, "error adding peer")
	}
}

func loadPrivateKeyFromFile(path string, passwd string) (*ecdsa.PrivateKey, error) {
	src, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var encryptedK keystore.CryptoJSON
	err = json.Unmarshal(src, &encryptedK)
	if err != nil {
		return nil, err
	}

	pKey, err := keystore.DecryptDataV3(encryptedK, passwd)
	if err != nil {
		return nil, err
	}

	return crypto.ToECDSA(pKey)
}

func checkForFileExistence(path string, overwrite bool) error {
	_, err := os.Stat(path)

	if err == nil && !overwrite {
		return fmt.Errorf("%s already exists. Use --overwrite to overwrite the file", path)
	}

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

func generatePrivateKey() ([]byte, error) {
	key, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}

	return key.D.Bytes(), nil
}

func writePrivateKeyToFile(path string, passwd []byte, overwrite bool) error {
	if err := checkForFileExistence(path, overwrite); err != nil {
		return err
	}

	key, err := generatePrivateKey()
	if err != nil {
		return err
	}

	encryptedK, err := keystore.EncryptDataV3(key, passwd, keystore.StandardScryptN, keystore.StandardScryptP)
	if err != nil {
		return err
	}

	output, err := json.Marshal(encryptedK)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path, output, 0600)
}

func getPrivKey(options Options) (*ecdsa.PrivateKey, error) {
	var prvKey *ecdsa.PrivateKey
	var err error

	if options.NodeKey != nil {
		prvKey = options.NodeKey
	} else {
		// TODO: once https://github.com/urfave/cli/issues/1272 is fixed, remove env variable logic
		keyString := os.Getenv("GOWAKU-NODEKEY")
		if keyString != "" {
			if prvKey, err = crypto.ToECDSA(common.FromHex(keyString)); err != nil {
				return nil, fmt.Errorf("error converting key into valid ecdsa key: %w", err)
			}
		} else {
			if _, err := os.Stat(options.KeyFile); err == nil {
				if prvKey, err = loadPrivateKeyFromFile(options.KeyFile, options.KeyPasswd); err != nil {
					return nil, fmt.Errorf("could not read keyfile: %w", err)
				}
			} else {
				if os.IsNotExist(err) {
					if prvKey, err = crypto.GenerateKey(); err != nil {
						return nil, fmt.Errorf("error generating key: %w", err)
					}
				} else {
					return nil, fmt.Errorf("could not read keyfile: %w", err)
				}
			}
		}
	}
	return prvKey, nil
}

func printListeningAddresses(ctx context.Context, nodeOpts []node.WakuNodeOption, options Options) {
	params := new(node.WakuNodeParameters)
	for _, opt := range nodeOpts {
		err := opt(params)
		if err != nil {
			panic(err)
		}
	}

	var libp2pOpts []config.Option
	libp2pOpts = append(libp2pOpts,
		params.Identity(),
		libp2p.ListenAddrs(params.MultiAddresses()...),
	)

	if options.Websocket.Secure {
		transports := libp2p.ChainOptions(
			libp2p.Transport(tcp.NewTCPTransport),
			libp2p.Transport(ws.New, ws.WithTLSConfig(params.TLSConfig())),
		)
		libp2pOpts = append(libp2pOpts, transports)
	}

	addrFactory := params.AddressFactory()
	if addrFactory != nil {
		libp2pOpts = append(libp2pOpts, libp2p.AddrsFactory(addrFactory))
	}

	h, err := libp2p.New(libp2pOpts...)
	if err != nil {
		panic(err)
	}

	hostInfo, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/p2p/%s", h.ID().Pretty()))

	for _, addr := range h.Addrs() {
		fmt.Println(addr.Encapsulate(hostInfo))
	}

}
