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
	"github.com/libp2p/go-libp2p-peerstore/pstoreds"
	"github.com/multiformats/go-multiaddr"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/status-im/go-waku/waku/metrics"
	"github.com/status-im/go-waku/waku/persistence"
	"github.com/status-im/go-waku/waku/persistence/sqlite"

	"github.com/status-im/go-waku/waku/v2/discovery"
	"github.com/status-im/go-waku/waku/v2/node"
	"github.com/status-im/go-waku/waku/v2/protocol/relay"

	pubsub "github.com/status-im/go-wakurelay-pubsub"
)

var log = logging.Logger("wakunode")

func randomHex(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func checkError(err error, msg string) {
	if err != nil {
		if msg != "" {
			msg = msg + ": "
		}
		log.Fatal(msg, err)
	}
}

var rootCmd = &cobra.Command{
	Use:   "waku",
	Short: "Start a waku node",
	Long:  `Start a waku node...`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetInt("port")
		enableWs, _ := cmd.Flags().GetBool("ws")
		wsPort, _ := cmd.Flags().GetInt("ws-port")
		wakuRelay, _ := cmd.Flags().GetBool("relay")
		wakuFilter, _ := cmd.Flags().GetBool("filter")
		key, _ := cmd.Flags().GetString("nodekey")
		store, _ := cmd.Flags().GetBool("store")
		useDB, _ := cmd.Flags().GetBool("use-db")
		dbPath, _ := cmd.Flags().GetString("dbpath")
		storenode, _ := cmd.Flags().GetString("storenode")
		staticnodes, _ := cmd.Flags().GetStringSlice("staticnodes")
		filternodes, _ := cmd.Flags().GetStringSlice("filternodes")
		lightpush, _ := cmd.Flags().GetBool("lightpush")
		lightpushnodes, _ := cmd.Flags().GetStringSlice("lightpushnodes")
		topics, _ := cmd.Flags().GetStringSlice("topics")
		keepAlive, _ := cmd.Flags().GetInt("keep-alive")
		enableMetrics, _ := cmd.Flags().GetBool("metrics")
		metricsAddress, _ := cmd.Flags().GetString("metrics-address")
		metricsPort, _ := cmd.Flags().GetInt("metrics-port")
		enableDnsDiscovery, _ := cmd.Flags().GetBool("dns-discovery")
		dnsDiscoveryUrl, _ := cmd.Flags().GetString("dns-discovery-url")
		dnsDiscoveryNameServer, _ := cmd.Flags().GetString("dns-discovery-nameserver")
		peerExchange, _ := cmd.Flags().GetBool("peer-exchange")

		hostAddr, _ := net.ResolveTCPAddr("tcp", fmt.Sprint("0.0.0.0:", port))

		var err error

		if key == "" {
			key, err = randomHex(32)
			checkError(err, "could not generate random key")
		}

		prvKey, err := crypto.HexToECDSA(key)
		checkError(err, "error converting key into valid ecdsa key")

		// TODO: this ENR record might be necessary later for DNS discovery
		// enr := enode.NewV4(&prvKey.PublicKey, hostAddr.IP, hostAddr.Port, 0)

		if dbPath == "" && useDB {
			checkError(errors.New("dbpath can't be null"), "")
		}

		var db *sql.DB

		if useDB {
			db, err = sqlite.NewDB(dbPath)
			checkError(err, "Could not connect to DB")
		}

		ctx := context.Background()

		var metricsServer *metrics.Server
		if enableMetrics {
			metricsServer = metrics.NewMetricsServer(metricsAddress, metricsPort)
			go metricsServer.Start()
		}

		nodeOpts := []node.WakuNodeOption{
			node.WithPrivateKey(prvKey),
			node.WithHostAddress([]net.Addr{hostAddr}),
			node.WithKeepAlive(time.Duration(keepAlive) * time.Second),
		}

		if enableWs {
			wsMa, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d/ws", wsPort))
			nodeOpts = append(nodeOpts, node.WithMultiaddress([]multiaddr.Multiaddr{wsMa}))
		}

		libp2pOpts := node.DefaultLibP2POptions

		if useDB {
			// Create persistent peerstore
			queries, err := sqlite.NewQueries("peerstore", db)
			checkError(err, "Peerstore")

			datastore := dssql.NewDatastore(db, queries)
			opts := pstoreds.DefaultOpts()
			peerStore, err := pstoreds.NewPeerstore(ctx, datastore, opts)
			checkError(err, "Peerstore")

			libp2pOpts = append(libp2pOpts, libp2p.Peerstore(peerStore))
		}

		nodeOpts = append(nodeOpts, node.WithLibP2POptions(libp2pOpts...))

		if wakuRelay {
			var wakurelayopts []pubsub.Option

			if peerExchange {
				wakurelayopts = append(wakurelayopts, pubsub.WithPeerExchange(true))
			}

			nodeOpts = append(nodeOpts, node.WithWakuRelay(wakurelayopts...))
		}

		if wakuFilter {
			nodeOpts = append(nodeOpts, node.WithWakuFilter())
		}

		if store {
			nodeOpts = append(nodeOpts, node.WithWakuStore(true, true))
			if useDB {
				dbStore, err := persistence.NewDBStore(persistence.WithDB(db))
				checkError(err, "DBStore")
				nodeOpts = append(nodeOpts, node.WithMessageProvider(dbStore))
			} else {
				nodeOpts = append(nodeOpts, node.WithMessageProvider(nil))
			}
		}

		if lightpush {
			nodeOpts = append(nodeOpts, node.WithLightPush())
		}

		wakuNode, err := node.New(ctx, nodeOpts...)

		checkError(err, "Wakunode")

		for _, t := range topics {
			nodeTopic := relay.Topic(t)
			_, err := wakuNode.Subscribe(&nodeTopic)
			checkError(err, "Error subscring to topic")
		}

		if storenode != "" && !store {
			checkError(errors.New("Store protocol was not started"), "")
		} else {
			if storenode != "" {
				_, err = wakuNode.AddStorePeer(storenode)
				if err != nil {
					log.Error("error adding store peer ", err)
				}
			}
		}

		if len(staticnodes) > 0 {
			for _, n := range staticnodes {
				go func(node string) {
					ctx, cancel := context.WithTimeout(ctx, time.Duration(3)*time.Second)
					defer cancel()
					err = wakuNode.DialPeer(ctx, node)
					if err != nil {
						log.Error("error dialing peer ", err)
					}
				}(n)
			}
		}

		if enableDnsDiscovery {
			for _, addr := range wakuNode.ListenAddresses() {
				ip, _ := addr.ValueForProtocol(multiaddr.P_IP4)
				enr := enode.NewV4(&prvKey.PublicKey, net.ParseIP(ip), hostAddr.Port, 0)
				log.Info("ENR: ", enr)
			}

			if dnsDiscoveryUrl != "" {
				log.Info("attempting DNS discovery with ", dnsDiscoveryUrl)
				multiaddresses, err := discovery.RetrieveNodes(ctx, dnsDiscoveryUrl, discovery.WithNameserver(dnsDiscoveryNameServer))
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

		if len(lightpushnodes) > 0 && !lightpush {
			checkError(errors.New("LightPush protocol was not started"), "")
		} else {
			if len(lightpushnodes) > 0 {
				for _, n := range lightpushnodes {
					go func(node string) {
						_, err = wakuNode.AddLightPushPeer(node)
						checkError(err, "Error adding lightpush peer")
					}(n)
				}
			}
		}

		if len(filternodes) > 0 && !wakuFilter {
			checkError(errors.New("WakuFilter protocol was not started"), "")
		} else {
			if len(filternodes) > 0 {
				for _, n := range filternodes {
					go func(node string) {
						_, err = wakuNode.AddFilterPeer(node)
						checkError(err, "Error adding filter peer")
					}(n)
				}
			}
		}

		// Wait for a SIGINT or SIGTERM signal
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		<-ch
		fmt.Println("\n\n\nReceived signal, shutting down...")

		// shut the node down
		wakuNode.Stop()

		if enableMetrics {
			metricsServer.Stop(ctx)
		}

		if useDB {
			err = db.Close()
			checkError(err, "DBClose")
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.Flags().Int("port", 9000, "Libp2p TCP listening port (0 for random)")
	rootCmd.Flags().Bool("ws", false, "Enable websockets support")
	rootCmd.Flags().Int("ws-port", 9001, "Libp2p TCP listening port for websocket connection (0 for random)")
	rootCmd.Flags().String("nodekey", "", "P2P node private key as hex (default random)")
	rootCmd.Flags().StringSlice("topics", []string{string(relay.DefaultWakuTopic)}, fmt.Sprintf("List of topics to listen (default %s)", relay.DefaultWakuTopic))
	rootCmd.Flags().StringSlice("staticnodes", []string{}, "Multiaddr of peer to directly connect with. Argument may be repeated")
	rootCmd.Flags().Bool("relay", true, "Enable relay protocol")
	rootCmd.Flags().Bool("filter", true, "Enable filter protocol")
	rootCmd.Flags().Bool("store", false, "Enable store protocol")
	rootCmd.Flags().Bool("lightpush", false, "Enable lightpush protocol")
	rootCmd.Flags().Bool("use-db", true, "Store messages and peers in a DB, (default: true, use false for in-memory only)")
	rootCmd.Flags().String("dbpath", "./store.db", "Path to DB file")
	rootCmd.Flags().String("storenode", "", "Multiaddr of peer to connect with for waku store protocol")
	rootCmd.Flags().Int("keep-alive", 300, "interval in seconds for pinging peers to keep the connection alive.")
	rootCmd.Flags().StringSlice("filternodes", []string{}, "Multiaddr of peers to to request content filtering of messages. Argument may be repeated")
	rootCmd.Flags().StringSlice("lightpushnodes", []string{}, "Multiaddr of peers to to request lightpush of published messages. Argument may be repeated")
	rootCmd.Flags().Bool("metrics", false, "Enable the metrics server")
	rootCmd.Flags().String("metrics-address", "127.0.0.1", "Listening address of the metrics server")
	rootCmd.Flags().Int("metrics-port", 8008, "Listening HTTP port of the metrics server")
	rootCmd.Flags().Bool("dns-discovery", false, "enable dns discovery")
	rootCmd.Flags().String("dns-discovery-url", "", "URL for DNS node list in format 'enrtree://<key>@<fqdn>'")
	rootCmd.Flags().String("dns-discovery-nameserver", "", "DNS nameserver IP to query (empty to use system's default)")
	rootCmd.Flags().Bool("peer-exchange", true, "Enable GossipSub Peer Exchange")
}

func initConfig() {
	viper.AutomaticEnv() // read in environment variables that match
}
