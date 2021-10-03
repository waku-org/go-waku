package waku

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
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
	libp2pdisc "github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/libp2p/go-libp2p-peerstore/pstoreds"
	"github.com/multiformats/go-multiaddr"
	rendezvous "github.com/status-im/go-libp2p-rendezvous"
	"github.com/status-im/go-waku/waku/metrics"
	"github.com/status-im/go-waku/waku/persistence"
	"github.com/status-im/go-waku/waku/persistence/sqlite"
	"github.com/status-im/go-waku/waku/v2/discovery"
	"github.com/status-im/go-waku/waku/v2/node"
	"github.com/status-im/go-waku/waku/v2/protocol/filter"
	"github.com/status-im/go-waku/waku/v2/protocol/lightpush"
	"github.com/status-im/go-waku/waku/v2/protocol/relay"
	"github.com/status-im/go-waku/waku/v2/protocol/store"
	pubsub "github.com/status-im/go-wakurelay-pubsub"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

var log = logging.Logger("wakunode")

func randomHex(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func failOnErr(err error, msg string) {
	if err != nil {
		if msg != "" {
			msg = msg + ": "
		}
		log.Fatal(msg, err)
	}
}

func Execute(options Options) {
	hostAddr, _ := net.ResolveTCPAddr("tcp", fmt.Sprint("0.0.0.0:", options.Port))

	var err error

	if options.NodeKey == "" {
		options.NodeKey, err = randomHex(32)
		failOnErr(err, "could not generate random key")
	}

	prvKey, err := crypto.HexToECDSA(options.NodeKey)
	failOnErr(err, "error converting key into valid ecdsa key")

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
		node.WithHostAddress([]net.Addr{hostAddr}),
		node.WithKeepAlive(time.Duration(options.KeepAlive) * time.Second),
	}

	if options.EnableWS {
		wsMa, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d/ws", options.WSPort))
		nodeOpts = append(nodeOpts, node.WithMultiaddress([]multiaddr.Multiaddr{wsMa}))
	}

	libp2pOpts := node.DefaultLibP2POptions

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
		db, err := leveldb.OpenFile(options.RendezvousServer.DBPath, &opt.Options{OpenFilesCacheCapacity: 3})
		failOnErr(err, "RendezvousDB")
		storage := rendezvous.NewStorage(db)
		nodeOpts = append(nodeOpts, node.WithRendezvousServer(storage))
	}

	if options.Filter.Enable {
		nodeOpts = append(nodeOpts, node.WithWakuFilter())
	}

	if options.Store.Enable {
		nodeOpts = append(nodeOpts, node.WithWakuStore(true, true))
		if options.UseDB {
			dbStore, err := persistence.NewDBStore(persistence.WithDB(db))
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
		nodeOpts = append(nodeOpts, node.WithRendezvous(pubsub.WithDiscoveryOpts(libp2pdisc.Limit(45), libp2pdisc.TTL(time.Duration(20)*time.Second))))
	}

	wakuNode, err := node.New(ctx, nodeOpts...)

	failOnErr(err, "Wakunode")

	if len(options.Relay.Topics) == 0 {
		options.Relay.Topics = []string{string(relay.DefaultWakuTopic)}
	}

	for _, t := range options.Relay.Topics {
		nodeTopic := relay.Topic(t)
		_, err := wakuNode.Subscribe(&nodeTopic)
		failOnErr(err, "Error subscring to topic")
	}

	addPeers(wakuNode, options.Rendezvous.Nodes, rendezvous.RendezvousID_v001)
	addPeers(wakuNode, options.Store.Nodes, store.StoreID_v20beta3)
	addPeers(wakuNode, options.LightPush.Nodes, lightpush.LightPushID_v20beta1)
	addPeers(wakuNode, options.Filter.Nodes, filter.FilterID_v20beta1)

	for _, n := range options.StaticNodes {
		go func(node string) {
			ctx, cancel := context.WithTimeout(ctx, time.Duration(3)*time.Second)
			defer cancel()
			err = wakuNode.DialPeer(ctx, node)
			if err != nil {
				log.Error("error dialing peer ", err)
			}
		}(n)
	}

	if options.DNSDiscovery.Enable {
		for _, addr := range wakuNode.ListenAddresses() {
			ip, _ := addr.ValueForProtocol(multiaddr.P_IP4)
			enr := enode.NewV4(&prvKey.PublicKey, net.ParseIP(ip), hostAddr.Port, 0)
			log.Info("ENR: ", enr)
		}

		if options.DNSDiscovery.URL != "" {
			log.Info("attempting DNS discovery with ", options.DNSDiscovery.URL)
			multiaddresses, err := discovery.RetrieveNodes(ctx, options.DNSDiscovery.URL, discovery.WithNameserver(options.DNSDiscovery.Nameserver))
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

	// Wait for a SIGINT or SIGTERM signal
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	fmt.Println("\n\n\nReceived signal, shutting down...")

	// shut the node down
	wakuNode.Stop()

	if options.Metrics.Enable {
		metricsServer.Stop(ctx)
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
