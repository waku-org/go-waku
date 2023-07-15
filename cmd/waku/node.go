package main

import (
	"context"
	"crypto/ecdsa"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
	"github.com/pbnjay/memory"

	"github.com/waku-org/go-waku/waku/persistence/sqlite"
	dbutils "github.com/waku-org/go-waku/waku/persistence/utils"
	wmetrics "github.com/waku-org/go-waku/waku/v2/metrics"
	"github.com/waku-org/go-waku/waku/v2/peers"
	"github.com/waku-org/go-waku/waku/v2/rendezvous"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enode"
	dssql "github.com/ipfs/go-ds-sql"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/config"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/p2p/host/peerstore/pstoreds"
	ws "github.com/libp2p/go-libp2p/p2p/transport/websocket"
	"github.com/multiformats/go-multiaddr"
	"github.com/waku-org/go-waku/logging"
	"github.com/waku-org/go-waku/waku/metrics"
	"github.com/waku-org/go-waku/waku/persistence"
	"github.com/waku-org/go-waku/waku/v2/dnsdisc"
	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-waku/waku/v2/protocol/filter"
	"github.com/waku-org/go-waku/waku/v2/protocol/legacy_filter"
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

func requiresDB(options Options) bool {
	return options.Store.Enable || options.Rendezvous.Server
}

func scalePerc(value float64) float64 {
	if value > 100 {
		return 100
	}

	if value < 0.1 {
		return 0.1
	}

	return value
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
	if requiresDB(options) {
		db, migrationFn, err = dbutils.ExtractDBAndMigration(options.Store.DatabaseURL)
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
	if len(options.AdvertiseAddresses) != 0 {
		nodeOpts = append(nodeOpts, node.WithAdvertiseAddresses(options.AdvertiseAddresses...))
	}

	if options.ExtIP != "" {
		ip := net.ParseIP(options.ExtIP)
		if ip == nil {
			failOnErr(errors.New("invalid IP address"), "could not set external IP address")
		}
		nodeOpts = append(nodeOpts, node.WithExternalIP(ip))
	}

	if options.DNS4DomainName != "" {
		nodeOpts = append(nodeOpts, node.WithDns4Domain(options.DNS4DomainName))
	}

	libp2pOpts := node.DefaultLibP2POptions

	memPerc := scalePerc(options.ResourceScalingMemoryPercent)
	fdPerc := scalePerc(options.ResourceScalingFDPercent)
	limits := rcmgr.DefaultLimits // Default memory limit: 1/8th of total memory, minimum 128MB, maximum 1GB
	scaledLimits := limits.Scale(int64(float64(memory.TotalMemory())*memPerc/100), int(float64(getNumFDs())*fdPerc/100))
	resourceManager, err := rcmgr.NewResourceManager(rcmgr.NewFixedLimiter(scaledLimits))
	failOnErr(err, "setting resource limits")

	libp2pOpts = append(libp2pOpts, libp2p.ResourceManager(resourceManager))
	libp2p.SetDefaultServiceLimits(&limits)

	if len(options.AdvertiseAddresses) == 0 {
		libp2pOpts = append(libp2pOpts, libp2p.NATPortMap()) // Attempt to open ports using uPNP for NATed hosts.)
	}

	// Node can be a circuit relay server
	if options.CircuitRelay {
		libp2pOpts = append(libp2pOpts, libp2p.EnableRelayService())
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

		nodeOpts = append(nodeOpts, node.WithPeerStore(peerStore))
	}

	nodeOpts = append(nodeOpts, node.WithLibP2POptions(libp2pOpts...))
	nodeOpts = append(nodeOpts, node.WithNTP())

	if options.Relay.Enable {
		var wakurelayopts []pubsub.Option
		wakurelayopts = append(wakurelayopts, pubsub.WithPeerExchange(options.Relay.PeerExchange))
		nodeOpts = append(nodeOpts, node.WithWakuRelayAndMinPeers(options.Relay.MinRelayPeersToPublish, wakurelayopts...))
	}

	nodeOpts = append(nodeOpts, node.WithWakuFilterLightNode())

	if options.Filter.Enable {
		nodeOpts = append(nodeOpts, node.WithWakuFilterFullNode(filter.WithTimeout(options.Filter.Timeout)))
	}

	if options.Filter.UseV1 {
		nodeOpts = append(nodeOpts, node.WithLegacyWakuFilter(!options.Filter.DisableFullNode, legacy_filter.WithTimeout(options.Filter.Timeout)))
	}

	var dbStore *persistence.DBStore
	if requiresDB(options) {
		dbStore, err = persistence.NewDBStore(logger,
			persistence.WithDB(db),
			persistence.WithMigrations(migrationFn), // TODO: refactor migrations out of DBStore, or merge DBStore with rendezvous DB
			persistence.WithRetentionPolicy(options.Store.RetentionMaxMessages, options.Store.RetentionTime),
		)
		failOnErr(err, "DBStore")
		nodeOpts = append(nodeOpts, node.WithMessageProvider(dbStore))
	}

	if options.Store.Enable {
		nodeOpts = append(nodeOpts, node.WithWakuStore())
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
					var discPeerInfo []peer.AddrInfo
					for _, n := range nodes {
						discPeerInfo = append(discPeerInfo, n.PeerInfo)
					}
					logger.Info("found dns entries ", zap.Any("nodes", discPeerInfo))
					discoveredNodes = append(discoveredNodes, nodes...)
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

	if options.Rendezvous.Enable {
		nodeOpts = append(nodeOpts, node.WithRendezvous(options.Rendezvous.Nodes))
	}

	if options.Rendezvous.Server {
		rdb := rendezvous.NewDB(ctx, db, logger)
		nodeOpts = append(nodeOpts, node.WithRendezvousServer(rdb))
	}

	checkForRLN(logger, options, &nodeOpts)

	wakuNode, err := node.New(nodeOpts...)

	utils.Logger().Info("Version details ", zap.String("version", node.Version), zap.String("commit", node.GitCommit))

	failOnErr(err, "Wakunode")

	if options.Filter.UseV1 {
		addStaticPeers(wakuNode, options.Filter.NodesV1, legacy_filter.FilterID_v20beta1)
	}

	if err = wakuNode.Start(ctx); err != nil {
		logger.Fatal("starting waku node", zap.Error(err))
	}

	for _, d := range discoveredNodes {
		wakuNode.Host().Peerstore().AddAddrs(d.PeerID, d.PeerInfo.Addrs, peerstore.PermanentAddrTTL)
	}

	addStaticPeers(wakuNode, options.Store.Nodes, store.StoreID_v20beta4)
	addStaticPeers(wakuNode, options.LightPush.Nodes, lightpush.LightPushID_v20beta1)
	addStaticPeers(wakuNode, options.Rendezvous.Nodes, rendezvous.RendezvousID)
	addStaticPeers(wakuNode, options.Filter.Nodes, filter.FilterSubscribeID_v20beta1)

	if len(options.Relay.Topics.Value()) == 0 {
		options.Relay.Topics = *cli.NewStringSlice(relay.DefaultWakuTopic)
	}

	if options.Relay.Enable {
		for _, nodeTopic := range options.Relay.Topics.Value() {
			nodeTopic := nodeTopic
			sub, err := wakuNode.Relay().SubscribeToTopic(ctx, nodeTopic)
			failOnErr(err, "Error subscring to topic")
			sub.Unsubscribe()

			if options.Rendezvous.Enable {
				// Register the node in rendezvous point
				// TODO: we have to determine how discovery would work with relay subscriptions.
				//       It might make sense to use pubsub.WithDiscovery option of gossipsub and
				//       register DiscV5, PeerExchange and Rendezvous. This should be an
				//       application concern instead of having (i.e. ./build/waku or the status app)
				//       instead of having the wakunode being the one deciding to advertise which
				//       topics/shards it supports
				wakuNode.Rendezvous().Register(ctx, nodeTopic)

				go func(nodeTopic string) {
					desiredOutDegree := wakuNode.Relay().Params().D
					t := time.NewTicker(7 * time.Second)
					defer t.Stop()
					for {
						select {
						case <-ctx.Done():
							return
						case <-t.C:
							peerCnt := len(wakuNode.Relay().PubSub().ListPeers(nodeTopic))
							peersToFind := desiredOutDegree - peerCnt
							if peersToFind <= 0 {
								continue
							}

							ctx, cancel := context.WithTimeout(ctx, 7*time.Second)
							wakuNode.Rendezvous().Discover(ctx, nodeTopic, peersToFind)
							cancel()
						}
					}
				}(nodeTopic)

			}
		}

		for _, protectedTopic := range options.Relay.ProtectedTopics {
			err := wakuNode.Relay().AddSignedTopicValidator(protectedTopic.Topic, protectedTopic.PublicKey)
			failOnErr(err, "Error adding signed topic validator")
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

	if options.DiscV5.Enable {
		if err = wakuNode.DiscV5().Start(ctx); err != nil {
			logger.Fatal("starting discovery v5", zap.Error(err))
		}
	}

	// retrieve and connect to peer exchange peers
	if options.PeerExchange.Enable && options.PeerExchange.Node != nil {
		logger.Info("retrieving peer info via peer exchange protocol")

		peerID, err := wakuNode.AddPeer(*options.PeerExchange.Node, peers.Static, peer_exchange.PeerExchangeID_v20alpha1)
		if err != nil {
			logger.Error("adding peer exchange peer", logging.MultiAddrs("node", *options.PeerExchange.Node), zap.Error(err))
		} else {
			desiredOutDegree := wakuNode.Relay().Params().D
			if err = wakuNode.PeerExchange().Request(ctx, desiredOutDegree, peer_exchange.WithPeer(peerID)); err != nil {
				logger.Error("requesting peers via peer exchange", zap.Error(err))
			}
		}
	}

	if len(discoveredNodes) != 0 {
		for _, n := range discoveredNodes {
			go func(ctx context.Context, info peer.AddrInfo) {
				ctx, cancel := context.WithTimeout(ctx, dialTimeout)
				defer cancel()
				err = wakuNode.DialPeerWithInfo(ctx, info)
				if err != nil {
					logger.Error("dialing peer", logging.HostID("peer", info.ID), zap.Error(err))
				}
			}(ctx, n.PeerInfo)

		}
	}

	var wg sync.WaitGroup

	if options.Store.Enable && len(options.Store.ResumeNodes) != 0 {
		// TODO: extract this to a function and run it when you go offline
		// TODO: determine if a store is listening to a topic

		var peerIDs []peer.ID
		for _, n := range options.Store.ResumeNodes {
			pID, err := wakuNode.AddPeer(n, peers.Static, store.StoreID_v20beta4)
			if err != nil {
				logger.Warn("adding peer to peerstore", logging.MultiAddrs("peer", n), zap.Error(err))
			}
			peerIDs = append(peerIDs, pID)
		}

		for _, t := range options.Relay.Topics.Value() {
			wg.Add(1)
			go func(topic string) {
				defer wg.Done()
				ctxWithTimeout, ctxCancel := context.WithTimeout(ctx, 20*time.Second)
				defer ctxCancel()
				if _, err := wakuNode.Store().Resume(ctxWithTimeout, topic, peerIDs); err != nil {
					logger.Error("Could not resume history", zap.Error(err))
				}
			}(t)
		}
	}

	var rpcServer *rpc.WakuRpc
	if options.RPCServer.Enable {
		rpcServer = rpc.NewWakuRpc(wakuNode, options.RPCServer.Address, options.RPCServer.Port, options.RPCServer.Admin, options.RPCServer.Private, options.PProf, options.RPCServer.RelayCacheCapacity, logger)
		rpcServer.Start()
	}

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

func addStaticPeers(wakuNode *node.WakuNode, addresses []multiaddr.Multiaddr, protocols ...protocol.ID) {
	for _, addr := range addresses {
		_, err := wakuNode.AddPeer(addr, peers.Static, protocols...)
		failOnErr(err, "error adding peer")
	}
}

func loadPrivateKeyFromFile(path string, passwd string) (*ecdsa.PrivateKey, error) {
	src, err := os.ReadFile(path)
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

	return os.WriteFile(path, output, 0600)
}

func getPrivKey(options Options) (*ecdsa.PrivateKey, error) {
	var prvKey *ecdsa.PrivateKey
	// get private key from nodeKey or keyFile
	if options.NodeKey != nil {
		prvKey = options.NodeKey
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

	hostAddrs := utils.EncapsulatePeerID(h.ID(), h.Addrs()...)
	for _, addr := range hostAddrs {
		fmt.Println(addr)
	}

}
