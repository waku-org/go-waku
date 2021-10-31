package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
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
	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/multiformats/go-multiaddr"
	"github.com/status-im/go-waku/waku/v2/discovery"
	"github.com/status-im/go-waku/waku/v2/node"
	"github.com/status-im/go-waku/waku/v2/protocol/filter"
	"github.com/status-im/go-waku/waku/v2/protocol/lightpush"
	"github.com/status-im/go-waku/waku/v2/protocol/store"
)

var DefaultContentTopic string = "/toy-chat/2/huilong/proto"

func main() {
	mrand.Seed(time.Now().UTC().UnixNano())

	nickFlag := flag.String("nick", "", "nickname to use in chat. will be generated if empty")
	fleetFlag := flag.String("fleet", "wakuv2.prod", "Select the fleet to connect to. (wakuv2.prod, wakuv2.test)")
	contentTopicFlag := flag.String("contenttopic", DefaultContentTopic, "content topic to use for the chat")
	nodeKeyFlag := flag.String("nodekey", "", "private key for this node. will be generated if empty")
	staticNodeFlag := flag.String("staticnode", "", "connects to a node. will get a random node from fleets.status.im if empty")
	relayFlag := flag.Bool("relay", true, "enable relay protocol")
	storeNodeFlag := flag.String("storenode", "", "connects to a store node to retrieve messages. will get a random node from fleets.status.im if empty")
	port := flag.Int("port", 0, "port. Will be random if 0")
	payloadV1Flag := flag.Bool("payloadV1", false, "use Waku v1 payload encoding/encryption. default false")
	filterFlag := flag.Bool("filter", false, "enable filter protocol")
	filterNodeFlag := flag.String("filternode", "", "multiaddr of peer to to request content filtering of messages")
	lightPushFlag := flag.Bool("lightpush", false, "enable lightpush protocol")
	lightPushNodeFlag := flag.String("lightpushnode", "", "Multiaddr of peer to to request lightpush of published messages")
	keepAliveFlag := flag.Int64("keep-alive", 20, "interval in seconds for pinging peers to keep the connection alive.")

	dnsDiscoveryFlag := flag.Bool("dns-discovery", false, "enable dns discovery")
	dnsDiscoveryUrlFlag := flag.String("dns-discovery-url", "", "URL for DNS node list in format 'enrtree://<key>@<fqdn>'")
	dnsDiscoveryNameServerFlag := flag.String("dns-discovery-nameserver", "", "DNS name server IP to query (empty to use system default)")

	flag.Parse()

	hostAddr, _ := net.ResolveTCPAddr("tcp", fmt.Sprintf("0.0.0.0:%d", *port))

	if *fleetFlag != "wakuv2.prod" && *fleetFlag != "wakuv2.test" {
		fmt.Println("Invalid fleet. Valid values are wakuv2.prod and wakuv2.test")
		return
	}

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

	opts := []node.WakuNodeOption{
		node.WithPrivateKey(prvKey),
		node.WithHostAddress([]*net.TCPAddr{hostAddr}),
		node.WithWakuStore(false, false),
		node.WithKeepAlive(time.Duration(*keepAliveFlag) * time.Second),
	}

	if *relayFlag {
		opts = append(opts, node.WithWakuRelay())
	}

	if *filterFlag {
		opts = append(opts, node.WithWakuFilter(false))
	}

	if *lightPushFlag || *lightPushNodeFlag != "" {
		*lightPushFlag = true // If a lightpushnode was set and lightpush flag was false
		opts = append(opts, node.WithLightPush())
	}

	wakuNode, err := node.New(ctx, opts...)
	if err != nil {
		fmt.Print(err)
		return
	}

	if *lightPushFlag {
		addPeer(wakuNode, *lightPushNodeFlag, lightpush.LightPushID_v20beta1)
	}

	if *filterFlag {
		addPeer(wakuNode, *filterNodeFlag, filter.FilterID_v20beta1)
	}

	if err := wakuNode.Start(); err != nil {
		panic(err)
	}

	// use the nickname from the cli flag, or a default if blank
	nick := *nickFlag
	if len(nick) == 0 {
		nick = defaultNick(wakuNode.Host().ID())
	}

	// join the chat
	chat, err := NewChat(ctx, wakuNode, wakuNode.Host().ID(), *contentTopicFlag, *payloadV1Flag, *lightPushFlag, nick)
	if err != nil {
		panic(err)
	}

	// Display panic level to reduce log noise
	lvl, err := logging.LevelFromString("panic")
	if err != nil {
		panic(err)
	}
	logging.SetAllLoggers(lvl)

	ui := NewChatUI(ctx, chat)

	// Connect to a static node or use random node from fleets.status.im
	go func() {
		time.Sleep(200 * time.Millisecond)

		staticnode := *staticNodeFlag
		storenode := *storeNodeFlag

		var fleetData []byte
		if len(staticnode) == 0 || len(storenode) == 0 {
			fleetData = getFleetData()
		}

		if len(staticnode) == 0 {
			ui.displayMessage(fmt.Sprintf("No static peers configured. Choosing one at random from %s fleet...", *fleetFlag))
			staticnode = getRandomFleetNode(fleetData, *fleetFlag)
		}

		ctx, cancel := context.WithTimeout(ctx, time.Duration(5)*time.Second)
		defer cancel()
		err = wakuNode.DialPeer(ctx, staticnode)
		if err != nil {
			ui.displayMessage("Could not connect to peer: " + err.Error())
			return
		} else {
			ui.displayMessage("Connected to peer: " + staticnode)
		}

		enableDiscovery := *dnsDiscoveryFlag
		dnsDiscoveryUrl := *dnsDiscoveryUrlFlag
		dnsDiscoveryNameServer := *dnsDiscoveryNameServerFlag

		if enableDiscovery && dnsDiscoveryUrl != "" {
			ui.displayMessage(fmt.Sprintf("attempting DNS discovery with %s", dnsDiscoveryUrl))
			multiaddresses, err := discovery.RetrieveNodes(ctx, dnsDiscoveryUrl, discovery.WithNameserver(dnsDiscoveryNameServer))
			if err != nil {
				ui.displayMessage("DNS discovery error: " + err.Error())
			} else {
				for _, m := range multiaddresses {
					go func(ctx context.Context, m multiaddr.Multiaddr) {
						ctx, cancel := context.WithTimeout(ctx, time.Duration(3)*time.Second)
						defer cancel()
						err = wakuNode.DialPeerWithMultiAddress(ctx, m)
						if err != nil {
							ui.displayMessage("error dialing peer: " + err.Error())
						}
					}(ctx, m)
				}
			}
		}

		if len(storenode) == 0 {
			ui.displayMessage(fmt.Sprintf("No store node configured. Choosing one at random from %s fleet...", *fleetFlag))
			storenode = getRandomFleetNode(fleetData, *fleetFlag)
		}

		storeNodeId, err := addPeer(wakuNode, storenode, store.StoreID_v20beta3)
		if err != nil {
			ui.displayMessage("Could not connect to storenode: " + err.Error())
			return
		} else {
			ui.displayMessage("Connected to storenode: " + storenode)
		}

		time.Sleep(300 * time.Millisecond)
		ui.displayMessage("Querying historic messages")

		tCtx, _ := context.WithTimeout(ctx, 5*time.Second)

		q := store.Query{
			ContentTopics: []string{*contentTopicFlag},
		}
		response, err := wakuNode.Query(tCtx, q,
			store.WithAutomaticRequestId(),
			store.WithPeer(*storeNodeId),
			store.WithPaging(true, 0))

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

	wakuNode.Stop()
	// TODO: filter unsubscribeAll

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

func getRandomFleetNode(data []byte, fleetId string) string {
	var result map[string]interface{}
	json.Unmarshal(data, &result)
	fleets := result["fleets"].(map[string]interface{})
	fleet := fleets[fleetId].(map[string]interface{})
	waku := fleet["waku"].(map[string]interface{})

	var wakunodes []string
	for v := range waku {
		wakunodes = append(wakunodes, v)
		break
	}

	randKey := wakunodes[mrand.Intn(len(wakunodes))]

	return waku[randKey].(string)
}

func addPeer(wakuNode *node.WakuNode, addr string, protocol protocol.ID) (*peer.ID, error) {
	if addr == "" {
		return nil, errors.New("invalid multiaddress")
	}

	ma, err := multiaddr.NewMultiaddr(addr)
	if err != nil {
		return nil, err
	}

	return wakuNode.AddPeer(ma, protocol)
}
