package waku

import (
	"context"
	"crypto/ecdsa"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enode"
	dssql "github.com/ipfs/go-ds-sql"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p/config"

	"github.com/libp2p/go-libp2p-peerstore/pstoreds"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/multiformats/go-multiaddr"
	rendezvous "github.com/status-im/go-waku-rendezvous"
	"github.com/status-im/go-waku/logging"
	"github.com/status-im/go-waku/waku/metrics"
	"github.com/status-im/go-waku/waku/persistence"
	"github.com/status-im/go-waku/waku/persistence/sqlite"
	"github.com/status-im/go-waku/waku/v2/dnsdisc"
	"github.com/status-im/go-waku/waku/v2/node"
	"github.com/status-im/go-waku/waku/v2/protocol/filter"
	"github.com/status-im/go-waku/waku/v2/protocol/lightpush"
	"github.com/status-im/go-waku/waku/v2/protocol/relay"
	"github.com/status-im/go-waku/waku/v2/protocol/store"
	"github.com/status-im/go-waku/waku/v2/rest"
	"github.com/status-im/go-waku/waku/v2/rpc"
	"github.com/status-im/go-waku/waku/v2/utils"
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

	if options.DBPath == "" && options.UseDB {
		failOnErr(errors.New("dbpath can't be null"), "")
	}

	var db *sql.DB
	if options.UseDB {
		db, err = sqlite.NewDB(options.DBPath)
		failOnErr(err, "Could not connect to DB")
		logger.Debug("using database: ", zap.String("path", options.DBPath))

	} else {
		db, err = sqlite.NewDB(":memory:")
		failOnErr(err, "Could not create in-memory DB")
		logger.Debug("using in-memory database")
	}

	ctx := context.Background()

	var metricsServer *metrics.Server
	if options.Metrics.Enable {
		metricsServer = metrics.NewMetricsServer(options.Metrics.Address, options.Metrics.Port, logger)
		go metricsServer.Start()
	}

	nodeOpts := []node.WakuNodeOption{
		node.WithLogger(logger),
		node.WithPrivateKey(prvKey),
		node.WithHostAddress(hostAddr),
		node.WithKeepAlive(time.Duration(options.KeepAlive) * time.Second),
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

	if options.Websocket.Enable {
		nodeOpts = append(nodeOpts, node.WithWebsockets(options.Websocket.Address, options.Websocket.Port))
	}

	if options.Websocket.Secure {
		nodeOpts = append(nodeOpts, node.WithSecureWebsockets(options.Websocket.Address, options.Websocket.Port, options.Websocket.CertPath, options.Websocket.KeyPath))
	}

	if options.ShowAddresses {
		printListeningAddresses(ctx, nodeOpts, options)
		return
	}

	if options.Version {
		fmt.Printf("version / git commit hash: %s-%s\n", node.Version, node.GitCommit)
		return
	}

	if options.UseDB {
		if options.PersistPeers {
			// Create persistent peerstore
			queries, err := sqlite.NewQueries("peerstore", db)
			failOnErr(err, "Peerstore")

			datastore := dssql.NewDatastore(db, queries)
			opts := pstoreds.DefaultOpts()
			peerStore, err := pstoreds.NewPeerstore(ctx, datastore, opts)
			failOnErr(err, "Peerstore")

			libp2pOpts = append(libp2pOpts, libp2p.Peerstore(peerStore))
		}
	}

	nodeOpts = append(nodeOpts, node.WithLibP2POptions(libp2pOpts...))

	if options.Relay.Enable {
		var wakurelayopts []pubsub.Option
		wakurelayopts = append(wakurelayopts, pubsub.WithPeerExchange(options.Relay.PeerExchange))
		nodeOpts = append(nodeOpts, node.WithWakuRelayAndMinPeers(options.Relay.MinRelayPeersToPublish, wakurelayopts...))
	}

	if options.RendezvousServer.Enable {
		db, err := persistence.NewRendezVousLevelDB(options.RendezvousServer.DBPath)
		failOnErr(err, "RendezvousDB")
		storage := rendezvous.NewStorage(db)
		nodeOpts = append(nodeOpts, node.WithRendezvousServer(storage))
	}

	if options.Filter.Enable {
		nodeOpts = append(nodeOpts, node.WithWakuFilter(!options.Filter.DisableFullNode, filter.WithTimeout(time.Duration(options.Filter.Timeout)*time.Second)))
	}

	if options.Store.Enable {
		if options.Store.PersistMessages {
			nodeOpts = append(nodeOpts, node.WithWakuStore(true, options.Store.ShouldResume))
			dbStore, err := persistence.NewDBStore(logger, persistence.WithDB(db), persistence.WithRetentionPolicy(options.Store.RetentionMaxMessages, options.Store.RetentionMaxSecondsDuration()))
			failOnErr(err, "DBStore")
			nodeOpts = append(nodeOpts, node.WithMessageProvider(dbStore))
		} else {
			nodeOpts = append(nodeOpts, node.WithWakuStore(false, false))
		}
	}

	if options.LightPush.Enable {
		nodeOpts = append(nodeOpts, node.WithLightPush())
	}

	if options.Rendezvous.Enable {
		nodeOpts = append(nodeOpts, node.WithRendezvous(pubsub.WithDiscoveryOpts(discovery.Limit(45), discovery.TTL(time.Duration(20)*time.Second))))
	}

	var discoveredNodes []dnsdisc.DiscoveredNode
	if options.DNSDiscovery.Enable {
		if options.DNSDiscovery.URL != "" {
			logger.Info("attempting DNS discovery with ", zap.String("URL", options.DNSDiscovery.URL))
			nodes, err := dnsdisc.RetrieveNodes(ctx, options.DNSDiscovery.URL, dnsdisc.WithNameserver(options.DNSDiscovery.Nameserver))
			if err != nil {
				logger.Warn("dns discovery error ", zap.Error(err))
			} else {
				logger.Info("found dns entries ", zap.Any("qty", len(nodes)))
				discoveredNodes = nodes
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

		nodeOpts = append(nodeOpts, node.WithDiscoveryV5(options.DiscV5.Port, bootnodes, options.DiscV5.AutoUpdate, pubsub.WithDiscoveryOpts(discovery.Limit(45), discovery.TTL(time.Duration(20)*time.Second))))
	}

	if options.RLNRelay.Enable {
		if !options.Relay.Enable {
			failOnErr(errors.New("relay not available"), "Could not enable RLN Relay")
		}

		if !options.RLNRelay.Dynamic {
			nodeOpts = append(nodeOpts, node.WithStaticRLNRelay(options.RLNRelay.PubsubTopic, options.RLNRelay.ContentTopic, rln.MembershipIndex(options.RLNRelay.MembershipIndex), nil))
		}
	}

	wakuNode, err := node.New(ctx, nodeOpts...)

	failOnErr(err, "Wakunode")

	addPeers(wakuNode, options.Rendezvous.Nodes.Value(), string(rendezvous.RendezvousID_v001))
	addPeers(wakuNode, options.Store.Nodes.Value(), string(store.StoreID_v20beta4))
	addPeers(wakuNode, options.LightPush.Nodes.Value(), string(lightpush.LightPushID_v20beta1))
	addPeers(wakuNode, options.Filter.Nodes.Value(), string(filter.FilterID_v20beta1))

	if err = wakuNode.Start(); err != nil {
		logger.Fatal("starting waku node", zap.Error(err))
	}

	if options.DiscV5.Enable {
		if err = wakuNode.DiscV5().Start(); err != nil {
			logger.Fatal("starting discovery v5", zap.Error(err))
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

	for _, n := range options.StaticNodes.Value() {
		go func(node string) {
			err = wakuNode.DialPeer(ctx, node)
			if err != nil {
				logger.Error("dialing peer", zap.Error(err))
			}
		}(n)
	}

	if len(discoveredNodes) != 0 {
		for _, n := range discoveredNodes {
			for _, m := range n.Addresses {
				go func(ctx context.Context, m multiaddr.Multiaddr) {
					ctx, cancel := context.WithTimeout(ctx, time.Duration(3)*time.Second)
					defer cancel()
					err = wakuNode.DialPeerWithMultiAddress(ctx, m)
					if err != nil {
						logger.Error("dialing peer ", zap.Error(err))
					}
				}(ctx, m)
			}
		}
	}

	var rpcServer *rpc.WakuRpc
	if options.RPCServer.Enable {
		rpcServer = rpc.NewWakuRpc(wakuNode, options.RPCServer.Address, options.RPCServer.Port, options.RPCServer.Admin, options.RPCServer.Private, logger)
		rpcServer.Start()
	}

	var restServer *rest.WakuRest
	if options.RESTServer.Enable {
		restServer = rest.NewWakuRest(wakuNode, options.RESTServer.Address, options.RESTServer.Port, options.RESTServer.Admin, options.RESTServer.Private, options.RESTServer.RelayCacheCapacity, logger)
		restServer.Start()
	}

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

	if options.UseDB {
		err = db.Close()
		failOnErr(err, "DBClose")
	}
}

func addPeers(wakuNode *node.WakuNode, addresses []string, protocols ...string) {
	for _, addrString := range addresses {
		if addrString == "" {
			continue
		}

		addr, err := multiaddr.NewMultiaddr(addrString)
		failOnErr(err, "invalid multiaddress")

		_, err = wakuNode.AddPeer(addr, protocols...)
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

func checkForPrivateKeyFile(path string, overwrite bool) error {
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
	if err := checkForPrivateKeyFile(path, overwrite); err != nil {
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
	if options.NodeKey != "" {
		if prvKey, err = crypto.HexToECDSA(options.NodeKey); err != nil {
			return nil, fmt.Errorf("error converting key into valid ecdsa key: %w", err)
		}
	} else {
		keyString := os.Getenv("GOWAKU-NODEKEY")
		if keyString != "" {
			if prvKey, err = crypto.HexToECDSA(keyString); err != nil {
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
