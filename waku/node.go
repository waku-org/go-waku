package waku

import (
	"context"
	"crypto/ecdsa"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enode"
	dssql "github.com/ipfs/go-ds-sql"

	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p"
	libp2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p/config"

	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/libp2p/go-libp2p-peerstore/pstoreds"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/multiformats/go-multiaddr"
	rendezvous "github.com/status-im/go-waku-rendezvous"
	"github.com/status-im/go-waku/waku/metrics"
	"github.com/status-im/go-waku/waku/persistence"
	"github.com/status-im/go-waku/waku/persistence/sqlite"
	"github.com/status-im/go-waku/waku/v2/dnsdisc"
	"github.com/status-im/go-waku/waku/v2/node"
	"github.com/status-im/go-waku/waku/v2/protocol/filter"
	"github.com/status-im/go-waku/waku/v2/protocol/lightpush"
	"github.com/status-im/go-waku/waku/v2/protocol/relay"
	"github.com/status-im/go-waku/waku/v2/protocol/store"
	"github.com/status-im/go-waku/waku/v2/rpc"
)

var log = logging.Logger("wakunode")

func failOnErr(err error, msg string) {
	if err != nil {
		if msg != "" {
			msg = msg + ": "
		}
		log.Fatal(msg, err)
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
		if err := writePrivateKeyToFile(options.KeyFile, options.Overwrite); err != nil {
			failOnErr(err, "nodekey error")
		}
		return
	}

	hostAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", options.Address, options.Port))
	failOnErr(err, "invalid host address")

	prvKey, err := getPrivKey(options)
	failOnErr(err, "nodekey error")

	if options.DBPath == "" && options.UseDB {
		failOnErr(errors.New("dbpath can't be null"), "")
	}

	var db *sql.DB

	if options.UseDB {
		db, err = sqlite.NewDB(options.DBPath)
		failOnErr(err, "Could not connect to DB")
	}

	ctx := context.Background()

	var metricsServer *metrics.Server
	if options.Metrics.Enable {
		metricsServer = metrics.NewMetricsServer(options.Metrics.Address, options.Metrics.Port)
		go metricsServer.Start()
	}

	nodeOpts := []node.WakuNodeOption{
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

		nodeOpts = append(nodeOpts, node.WithAdvertiseAddress(advertiseAddr, options.EnableWS, options.WSPort))
	}

	if options.EnableWS {
		wsMa, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d/ws", options.WSAddress, options.WSPort))
		nodeOpts = append(nodeOpts, node.WithMultiaddress([]multiaddr.Multiaddr{wsMa}))
	}

	if options.ShowAddresses {
		printListeningAddresses(ctx, nodeOpts, options)
		return
	}

	libp2pOpts := node.DefaultLibP2POptions
	if options.AdvertiseAddress == "" {
		libp2pOpts = append(libp2pOpts, libp2p.NATPortMap()) // Attempt to open ports using uPNP for NATed hosts.)
	}

	if options.UseDB {
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

	if !options.Relay.Disable {
		var wakurelayopts []pubsub.Option
		wakurelayopts = append(wakurelayopts, pubsub.WithPeerExchange(options.Relay.PeerExchange))
		nodeOpts = append(nodeOpts, node.WithWakuRelay(wakurelayopts...))
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
		nodeOpts = append(nodeOpts, node.WithWakuStoreAndRetentionPolicy(options.Store.ShouldResume, options.Store.RetentionMaxDaysDuration(), options.Store.RetentionMaxMessages))
		if options.UseDB {
			dbStore, err := persistence.NewDBStore(persistence.WithDB(db), persistence.WithRetentionPolicy(options.Store.RetentionMaxMessages, options.Store.RetentionMaxDaysDuration()))
			failOnErr(err, "DBStore")
			nodeOpts = append(nodeOpts, node.WithMessageProvider(dbStore))
		} else {
			nodeOpts = append(nodeOpts, node.WithMessageProvider(nil))
		}
	}

	if options.LightPush.Enable {
		nodeOpts = append(nodeOpts, node.WithLightPush())
	}

	if options.Rendezvous.Enable {
		nodeOpts = append(nodeOpts, node.WithRendezvous(pubsub.WithDiscoveryOpts(discovery.Limit(45), discovery.TTL(time.Duration(20)*time.Second))))
	}

	if options.DiscV5.Enable {
		var bootnodes []*enode.Node
		for _, addr := range options.DiscV5.Nodes {
			bootnode, err := enode.Parse(enode.ValidSchemes, addr)
			if err != nil {
				log.Fatal("could not parse enr: ", err)
			}
			bootnodes = append(bootnodes, bootnode)
		}
		nodeOpts = append(nodeOpts, node.WithDiscoveryV5(options.DiscV5.Port, bootnodes, options.DiscV5.AutoUpdate, pubsub.WithDiscoveryOpts(discovery.Limit(45), discovery.TTL(time.Duration(20)*time.Second))))
	}

	wakuNode, err := node.New(ctx, nodeOpts...)

	failOnErr(err, "Wakunode")

	addPeers(wakuNode, options.Rendezvous.Nodes, rendezvous.RendezvousID_v001)
	addPeers(wakuNode, options.Store.Nodes, store.StoreID_v20beta3)
	addPeers(wakuNode, options.LightPush.Nodes, lightpush.LightPushID_v20beta1)
	addPeers(wakuNode, options.Filter.Nodes, filter.FilterID_v20beta1)

	if err = wakuNode.Start(); err != nil {
		log.Fatal(fmt.Errorf("could not start waku node, %w", err))
	}

	if options.DiscV5.Enable {
		if err = wakuNode.DiscV5().Start(); err != nil {
			log.Fatal(fmt.Errorf("could not start discovery v5, %w", err))
		}
	}

	if len(options.Relay.Topics) == 0 {
		options.Relay.Topics = []string{string(relay.DefaultWakuTopic)}
	}

	if !options.Relay.Disable {
		for _, nodeTopic := range options.Relay.Topics {
			_, err := wakuNode.Relay().SubscribeToTopic(ctx, nodeTopic)
			failOnErr(err, "Error subscring to topic")
		}
	}

	for _, n := range options.StaticNodes {
		go func(node string) {
			err = wakuNode.DialPeer(ctx, node)
			if err != nil {
				log.Error("error dialing peer ", err)
			}
		}(n)
	}

	if options.DNSDiscovery.Enable {
		if options.DNSDiscovery.URL != "" {
			log.Info("attempting DNS discovery with ", options.DNSDiscovery.URL)
			multiaddresses, err := dnsdisc.RetrieveNodes(ctx, options.DNSDiscovery.URL, dnsdisc.WithNameserver(options.DNSDiscovery.Nameserver))
			if err != nil {
				log.Warn("dns discovery error ", err)
			} else {
				log.Info("found dns entries ", multiaddresses)
				for _, m := range multiaddresses {
					go func(ctx context.Context, m multiaddr.Multiaddr) {
						ctx, cancel := context.WithTimeout(ctx, time.Duration(3)*time.Second)
						defer cancel()
						err = wakuNode.DialPeerWithMultiAddress(ctx, m)
						if err != nil {
							log.Error("error dialing peer ", err)
						}
					}(ctx, m)
				}
			}
		} else {
			log.Fatal("DNS discovery URL is required")
		}
	}

	var rpcServer *rpc.WakuRpc
	if options.RPCServer.Enable {
		rpcServer = rpc.NewWakuRpc(wakuNode, options.RPCServer.Address, options.RPCServer.Port)
		rpcServer.Start()
	}

	// Wait for a SIGINT or SIGTERM signal
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	log.Info("Received signal, shutting down...")

	// shut the node down
	wakuNode.Stop()

	if options.RPCServer.Enable {
		err := rpcServer.Stop(ctx)
		failOnErr(err, "RPCClose")
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

func addPeers(wakuNode *node.WakuNode, addresses []string, protocol protocol.ID) {
	for _, addrString := range addresses {
		if addrString == "" {
			continue
		}

		addr, err := multiaddr.NewMultiaddr(addrString)
		failOnErr(err, "invalid multiaddress")

		_, err = wakuNode.AddPeer(addr, protocol)
		failOnErr(err, "error adding peer")
	}
}

func loadPrivateKeyFromFile(path string) (*ecdsa.PrivateKey, error) {
	src, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	dst := make([]byte, hex.DecodedLen(len(src)))
	_, err = hex.Decode(dst, src)
	if err != nil {
		return nil, err
	}

	p, err := libp2pcrypto.UnmarshalSecp256k1PrivateKey(dst)
	if err != nil {
		return nil, err
	}

	privKey := (*ecdsa.PrivateKey)(p.(*libp2pcrypto.Secp256k1PrivateKey))
	privKey.Curve = crypto.S256()

	return privKey, nil
}

func writePrivateKeyToFile(path string, force bool) error {
	_, err := os.Stat(path)

	if err == nil && !force {
		return fmt.Errorf("%s already exists. Use --overwrite to overwrite the file", path)
	}

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}

	key, err := crypto.GenerateKey()
	if err != nil {
		return err
	}

	privKey := libp2pcrypto.PrivKey((*libp2pcrypto.Secp256k1PrivateKey)(key))

	b, err := privKey.Raw()
	if err != nil {
		return err
	}

	output := make([]byte, hex.EncodedLen(len(b)))
	hex.Encode(output, b)

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
				if prvKey, err = loadPrivateKeyFromFile(options.KeyFile); err != nil {
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
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
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

	h, err := libp2p.New(ctx, libp2pOpts...)
	if err != nil {
		panic(err)
	}

	hostInfo, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/p2p/%s", h.ID().Pretty()))

	for _, addr := range h.Addrs() {
		fmt.Println(addr.Encapsulate(hostInfo))
	}

}
