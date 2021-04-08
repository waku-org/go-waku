package cmd

import (
	"context"
	"crypto/rand"
	"encoding/binary"
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
	"github.com/status-im/go-waku/waku/v2/protocol"
	store "github.com/status-im/go-waku/waku/v2/protocol/waku_store"
)

func randomHex(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

type DBStore struct {
	store.MessageProvider
}

func (dbStore *DBStore) Put(message *protocol.WakuMessage) error {
	fmt.Println("TODO: Implement MessageProvider.Put")
	return nil
}

func (dbStore *DBStore) GetAll() ([]*protocol.WakuMessage, error) {
	fmt.Println("TODO: Implement MessageProvider.GetAll. Returning a sample message")
	exampleMessage := new(protocol.WakuMessage)
	exampleMessage.ContentTopic = 1
	exampleMessage.Payload = []byte("Hello!")
	exampleMessage.Version = 0

	return []*protocol.WakuMessage{exampleMessage}, nil
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
		storenode, _ := cmd.Flags().GetString("storenode")
		staticnodes, _ := cmd.Flags().GetStringSlice("staticnodes")
		query, _ := cmd.Flags().GetBool("query")

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

		if store {
			err := wakuNode.MountStore(new(DBStore))
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

		if query {
			if !store {
				fmt.Println("Store protocol was not started")
				return
			}

			var DefaultContentTopic uint32 = binary.LittleEndian.Uint32([]byte("dingpu"))

			response, err := wakuNode.Query(DefaultContentTopic, true, 10)
			if err != nil {
				fmt.Println(err)
				return
			}

			fmt.Println(fmt.Sprint("Page Size: ", response.PagingInfo.PageSize))
			fmt.Println(fmt.Sprint("Direction: ", response.PagingInfo.Direction))
			if response.PagingInfo.Cursor != nil {
				fmt.Println(fmt.Sprint("Cursor - ReceivedTime: ", response.PagingInfo.Cursor.ReceivedTime))
				fmt.Println(fmt.Sprint("Cursor - Digest: ", hex.EncodeToString(response.PagingInfo.Cursor.Digest)))
			}
			fmt.Println("Messages:")
			for i, msg := range response.Messages {
				fmt.Println(fmt.Sprint(i, "- ", string(msg.Payload))) // Normaly you'd have to decode these, but i'm using v0
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
	rootCmd.Flags().String("storenode", "", "Multiaddr of peer to connect with for waku store protocol")
	rootCmd.Flags().Bool("relay", true, "Enable relay protocol")
	rootCmd.Flags().Bool("query", false, "Asks the storenode for stored messages")

}

func initConfig() {
	viper.AutomaticEnv() // read in environment variables that match
}
