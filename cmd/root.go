package cmd

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/status-im/go-waku/waku/v2/node"
)

func randomHex(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

var rootCmd = &cobra.Command{
	Use:   "waku",
	Short: "Start a waku node",
	Long:  `Start a waku node...`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetInt("port")
		relay, _ := cmd.Flags().GetBool("relay")
		key, _ := cmd.Flags().GetString("nodekey")
		store, _ := cmd.Flags().GetBool("store")
		dbPath, _ := cmd.Flags().GetString("dbpath")
		storenode, _ := cmd.Flags().GetString("storenode")
		staticnodes, _ := cmd.Flags().GetStringSlice("staticnodes")

		hostAddr, _ := net.ResolveTCPAddr("tcp", fmt.Sprint("0.0.0.0:", port))

		if key == "" {
			var err error
			key, err = randomHex(32)
			if err != nil {
				fmt.Println("Could not generate random key")
				return
			}
		}

		prvKey, err := crypto.HexToECDSA(key)

		ctx := context.Background()
		wakuNode, err := node.New(ctx, prvKey, []net.Addr{hostAddr})
		if err != nil {
			fmt.Println(err)
			return
		}

		if relay {
			wakuNode.MountRelay()
		}

		if store && dbPath != "" {
			db, err := NewDBStore(dbPath)
			if err != nil {
				fmt.Println(err)
				return
			}

			err = wakuNode.MountStore(db)
			if err != nil {
				fmt.Println(err)
				return
			}
			wakuNode.StartStore()
		}

		if storenode != "" && !store {
			fmt.Println("Store protocol was not started")
			return
		} else {
			if storenode != "" {
				wakuNode.AddStorePeer(storenode)
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
	rootCmd.Flags().String("nodekey", "", "P2P node private key as hex (default random)")
	rootCmd.Flags().StringSlice("staticnodes", []string{}, "Multiaddr of peer to directly connect with. Argument may be repeated")
	rootCmd.Flags().Bool("store", false, "Enable store protocol")
	rootCmd.Flags().String("dbpath", "./store.db", "Path to DB file")
	rootCmd.Flags().String("storenode", "", "Multiaddr of peer to connect with for waku store protocol")
	rootCmd.Flags().Bool("relay", true, "Enable relay protocol")

}

func initConfig() {
	viper.AutomaticEnv() // read in environment variables that match
}
