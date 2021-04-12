package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	mrand "math/rand"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/status-im/go-waku/waku/v2/node"
)

var DefaultContentTopic string = "dingpu"

func main() {
	mrand.Seed(time.Now().UTC().UnixNano())

	nickFlag := flag.String("nick", "", "nickname to use in chat. will be generated if empty")
	nodeKeyFlag := flag.String("nodekey", "", "private key for this node. will be generated if empty")
	staticNodeFlag := flag.String("staticnode", "", "connects to a node. will get a random node from fleets.status.im if empty")
	storeNodeFlag := flag.String("storenode", "", "connects to a store node to retrieve messages. will get a random node from fleets.status.im if empty")
	port := flag.Int("port", 0, "port. Will be random if 0")

	flag.Parse()

	hostAddr, _ := net.ResolveTCPAddr("tcp", fmt.Sprintf("0.0.0.0:%d", *port))

	// use the nickname from the cli flag, or a default if blank
	nodekey := *nodeKeyFlag
	if len(nodekey) == 0 {
		var err error
		nodekey, err = randomHex(32)
		if err != nil {
			fmt.Println("Could not generate random key")
			return
		}
	}
	prvKey, err := crypto.HexToECDSA(nodekey)

	ctx := context.Background()
	wakuNode, err := node.New(ctx, prvKey, []net.Addr{hostAddr})
	if err != nil {
		fmt.Print(err)
		return
	}

	wakuNode.MountRelay()
	wakuNode.MountStore(nil)

	// use the nickname from the cli flag, or a default if blank
	nick := *nickFlag
	if len(nick) == 0 {
		nick = defaultNick(wakuNode.Host().ID())
	}

	// join the chat
	chat, err := NewChat(ctx, wakuNode, wakuNode.Host().ID(), nick)
	if err != nil {
		panic(err)
	}

	ui := NewChatUI(chat)

	// Connect to a static node or use random node from fleets.status.im
	go func() {
		time.Sleep(200 * time.Millisecond)

		staticnode := *staticNodeFlag
		storenode := *storeNodeFlag
		//storenode = "/ip4/104.154.239.128/tcp/30303/p2p/16Uiu2HAmJb2e28qLXxT5kZxVUUoJt72EMzNGXB47Rxx5hw3q4YjS"
		storenode = "/ip4/127.0.0.1/tcp/60000/p2p/16Uiu2HAmULUvzjx7r5jcD8mtw6eGynjtAaDYUHFFbkbgcc3GNetZ"
		var fleetData []byte
		if len(staticnode) == 0 || len(storenode) == 0 {
			fleetData = getFleetData()
		}

		if len(staticnode) == 0 {
			ui.displayMessage("No static peers configured. Choosing one at random from test fleet...")
			staticnode = getRandomFleetNode(fleetData)
		}

		err = wakuNode.DialPeer(staticnode)
		if err != nil {
			ui.displayMessage("Could not connect to peer: " + err.Error())
			return
		} else {
			ui.displayMessage("Connected to peer: " + staticnode)

		}

		if len(storenode) == 0 {
			ui.displayMessage("No store node configured. Choosing one at random from test fleet...")
			storenode = getRandomFleetNode(fleetData)
		}

		err := wakuNode.AddStorePeer(storenode)
		if err != nil {
			ui.displayMessage("Could not connect to storenode: " + err.Error())
			return
		} else {
			ui.displayMessage("Connected to storenode: " + storenode)
		}

		time.Sleep(300 * time.Millisecond)
		ui.displayMessage("Querying historic messages")
		response, err := wakuNode.Query(DefaultContentTopic, true, 0)
		if err != nil {
			ui.displayMessage("Could not query storenode: " + err.Error())
		} else {
			chat.displayMessages(response.Messages)
		}
	}()

	//draw the UI
	if err = ui.Run(); err != nil {
		printErr("error running text UI: %s", err)
	}
}

// Generates a random hex string with a length of n
func randomHex(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// printErr is like fmt.Printf, but writes to stderr.
func printErr(m string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, m, args...)
}

// defaultNick generates a nickname based on the $USER environment variable and
// the last 8 chars of a peer ID.
func defaultNick(p peer.ID) string {
	return fmt.Sprintf("%s-%s", os.Getenv("USER"), shortID(p))
}

// shortID returns the last 8 chars of a base58-encoded peer id.
func shortID(p peer.ID) string {
	pretty := p.Pretty()
	return pretty[len(pretty)-8:]
}

func getFleetData() []byte {
	url := "https://fleets.status.im"
	httpClient := http.Client{
		Timeout: time.Second * 2,
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Fatal(err)
	}

	res, getErr := httpClient.Do(req)
	if getErr != nil {
		log.Fatal(getErr)
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		log.Fatal(readErr)
	}

	return body
}

func getRandomFleetNode(data []byte) string {
	var result map[string]interface{}
	json.Unmarshal(data, &result)

	fleets := result["fleets"].(map[string]interface{})
	wakuv2Test := fleets["wakuv2.test"].(map[string]interface{})
	waku := wakuv2Test["waku"].(map[string]interface{})

	var wakunodes []string
	for v := range waku {
		wakunodes = append(wakunodes, v)
		break
	}

	randKey := wakunodes[mrand.Intn(len(wakunodes))]

	return waku[randKey].(string)
}
