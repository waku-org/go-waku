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

	"github.com/ethereum/go-ethereum/crypto"
	dssql "github.com/ipfs/go-ds-sql"
	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-peerstore/pstoreds"
	"github.com/multiformats/go-multiaddr"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/status-im/go-waku/waku/persistence"
	"github.com/status-im/go-waku/waku/persistence/sqlite"

	"github.com/status-im/go-waku/waku/v2/node"
	"github.com/status-im/go-waku/waku/v2/protocol/relay"
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
		topics, _ := cmd.Flags().GetStringSlice("topics")

		hostAddr, _ := net.ResolveTCPAddr("tcp", fmt.Sprint("0.0.0.0:", port))

		var err error

		if key == "" {
			key, err = randomHex(32)
			checkError(err, "Could not generate random key")
		}

		prvKey, err := crypto.HexToECDSA(key)

		if dbPath == "" && useDB {
			checkError(errors.New("dbpath can't be null"), "")
		}

		var db *sql.DB

		if useDB {
			db, err = sqlite.NewDB(dbPath)
			checkError(err, "Could not connect to DB")
		}

		ctx := context.Background()

		nodeOpts := []node.WakuNodeOption{
			node.WithPrivateKey(prvKey),
			node.WithHostAddress([]net.Addr{hostAddr}),
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
			nodeOpts = append(nodeOpts, node.WithWakuRelay())
		}

		if wakuFilter {
			nodeOpts = append(nodeOpts, node.WithWakuFilter())
		}

		if store {
			nodeOpts = append(nodeOpts, node.WithWakuStore(true))
			if useDB {
				dbStore, err := persistence.NewDBStore(persistence.WithDB(db))
				checkError(err, "DBStore")
				nodeOpts = append(nodeOpts, node.WithMessageProvider(dbStore))
			} else {
				nodeOpts = append(nodeOpts, node.WithMessageProvider(nil))
			}
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
				checkError(err, "Error adding store peer")

			}
		}

		if len(staticnodes) > 0 {
			for _, n := range staticnodes {
				go wakuNode.DialPeer(n)
			}
		}

		// Wait for a SIGINT or SIGTERM signal
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		<-ch
		fmt.Println("\n\n\nReceived signal, shutting down...")

		// shut the node down
		wakuNode.Stop()

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
	rootCmd.Flags().Bool("use-db", true, "Store messages and peers in a DB, (default: true, use false for in-memory only)")
	rootCmd.Flags().String("dbpath", "./store.db", "Path to DB file")
	rootCmd.Flags().String("storenode", "", "Multiaddr of peer to connect with for waku store protocol")
}

func initConfig() {
	viper.AutomaticEnv() // read in environment variables that match
}
