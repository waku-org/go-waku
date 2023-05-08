package main

import (
	"chat2/pb"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/waku-org/go-waku/waku/v2/dnsdisc"
	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-waku/waku/v2/payload"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/filter"
	"github.com/waku-org/go-waku/waku/v2/protocol/legacy_filter"
	"github.com/waku-org/go-waku/waku/v2/protocol/lightpush"
	wpb "github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/protocol/store"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"github.com/waku-org/go-zerokit-rln/rln"
	"golang.org/x/crypto/pbkdf2"
	"google.golang.org/protobuf/proto"
)

// Chat represents a subscription to a single PubSub topic. Messages
// can be published to the topic with Chat.Publish, and received
// messages are pushed to the Messages channel.
type Chat struct {
	ctx       context.Context
	wg        sync.WaitGroup
	node      *node.WakuNode
	ui        UI
	uiReady   chan struct{}
	inputChan chan string
	options   Options

	C chan *protocol.Envelope

	nick string
}

func NewChat(ctx context.Context, node *node.WakuNode, options Options) *Chat {
	chat := &Chat{
		ctx:       ctx,
		node:      node,
		options:   options,
		nick:      options.Nickname,
		uiReady:   make(chan struct{}, 1),
		inputChan: make(chan string, 100),
	}

	chat.ui = NewUIModel(chat.uiReady, chat.inputChan)

	if options.Filter.Enable {
		if options.Filter.UseV2 {
			cf := filter.ContentFilter{
				Topic:         relay.DefaultWakuTopic,
				ContentTopics: []string{options.ContentTopic},
			}
			var filterOpt filter.FilterSubscribeOption
			peerID, err := options.Filter.NodePeerID()
			if err != nil {
				filterOpt = filter.WithAutomaticPeerSelection()
			} else {
				filterOpt = filter.WithPeer(peerID)
				chat.ui.InfoMessage(fmt.Sprintf("Subscribing to filter node %s", peerID))
			}
			theFilter, err := node.FilterLightnode().Subscribe(ctx, cf, filterOpt)
			if err != nil {
				chat.ui.ErrorMessage(err)
			} else {
				chat.C = theFilter.C
			}
		} else {
			// TODO: remove
			cf := legacy_filter.ContentFilter{
				Topic:         relay.DefaultWakuTopic,
				ContentTopics: []string{options.ContentTopic},
			}
			var filterOpt legacy_filter.FilterSubscribeOption
			peerID, err := options.Filter.NodePeerID()
			if err != nil {
				filterOpt = legacy_filter.WithAutomaticPeerSelection()
			} else {
				filterOpt = legacy_filter.WithPeer(peerID)
				chat.ui.InfoMessage(fmt.Sprintf("Subscribing to filter node %s", peerID))
			}
			_, theFilter, err := node.LegacyFilter().Subscribe(ctx, cf, filterOpt)
			if err != nil {
				chat.ui.ErrorMessage(err)
			} else {
				chat.C = theFilter.Chan
			}
		}

	} else {
		sub, err := node.Relay().Subscribe(ctx)
		if err != nil {
			chat.ui.ErrorMessage(err)
		} else {
			chat.C = make(chan *protocol.Envelope)
			go func() {
				for e := range sub.Ch {
					chat.C <- e
				}
			}()
		}
	}

	chat.wg.Add(6)
	go chat.parseInput()
	go chat.receiveMessages()

	connectionWg := sync.WaitGroup{}
	connectionWg.Add(2)

	go chat.welcomeMessage()

	go chat.staticNodes(&connectionWg)
	go chat.discoverNodes(&connectionWg)
	go chat.retrieveHistory(&connectionWg)

	return chat
}

func (c *Chat) Stop() {
	c.wg.Wait()
	close(c.inputChan)
}

func (c *Chat) receiveMessages() {
	defer c.wg.Done()
	for {
		select {
		case <-c.ctx.Done():
			return
		case value := <-c.C:

			msgContentTopic := value.Message().ContentTopic
			if msgContentTopic != c.options.ContentTopic || (c.options.RLNRelay.Enable && msgContentTopic != c.options.RLNRelay.ContentTopic) {
				continue // Discard messages from other topics
			}

			msg, err := decodeMessage(c.options.UsePayloadV1, c.options.ContentTopic, value.Message())
			if err == nil {
				// send valid messages to the UI
				c.ui.ChatMessage(int64(msg.Timestamp), msg.Nick, string(msg.Payload))
			}
		}
	}
}
func (c *Chat) parseInput() {
	defer c.wg.Done()
	for {
		select {
		case <-c.ctx.Done():
			return
		case line := <-c.inputChan:
			c.ui.SetSending(true)
			go func() {
				defer c.ui.SetSending(false)

				// bail if requested
				if line == "/exit" {
					c.ui.Quit()
					fmt.Println("Bye!")
					return
				}

				// add peer
				if strings.HasPrefix(line, "/connect") {
					peer := strings.TrimPrefix(line, "/connect ")
					c.wg.Add(1)
					go func(peer string) {
						defer c.wg.Done()

						ma, err := multiaddr.NewMultiaddr(peer)
						if err != nil {
							c.ui.ErrorMessage(err)
							return
						}

						peerID, err := ma.ValueForProtocol(multiaddr.P_P2P)
						if err != nil {
							c.ui.ErrorMessage(err)
							return
						}

						c.ui.InfoMessage(fmt.Sprintf("Connecting to peer: %s", peerID))
						ctx, cancel := context.WithTimeout(c.ctx, time.Duration(10)*time.Second)
						defer cancel()

						err = c.node.DialPeerWithMultiAddress(ctx, ma)
						if err != nil {
							c.ui.ErrorMessage(err)
						} else {
							c.ui.InfoMessage(fmt.Sprintf("Connected to %s", peerID))
						}
					}(peer)
					return
				}

				// list peers
				if line == "/peers" {
					peers := c.node.Host().Network().Peers()
					if len(peers) == 0 {
						c.ui.InfoMessage("No peers available")
					} else {
						peerInfoMsg := "Peers: \n"
						for _, p := range peers {
							peerInfo := c.node.Host().Peerstore().PeerInfo(p)
							peerProtocols, err := c.node.Host().Peerstore().GetProtocols(p)
							if err != nil {
								c.ui.ErrorMessage(err)
								return
							}
							peerInfoMsg += fmt.Sprintf("• %s:\n", p.Pretty())

							var strProtocols []string
							for _, p := range peerProtocols {
								strProtocols = append(strProtocols, string(p))
							}

							peerInfoMsg += fmt.Sprintf("    Protocols: %s\n", strings.Join(strProtocols, ", "))
							peerInfoMsg += "    Addresses:\n"
							for _, addr := range peerInfo.Addrs {
								peerInfoMsg += fmt.Sprintf("    - %s/p2p/%s\n", addr.String(), p.Pretty())
							}
						}
						c.ui.InfoMessage(peerInfoMsg)
					}
					return
				}

				// change nick
				if strings.HasPrefix(line, "/nick") {
					newNick := strings.TrimSpace(strings.TrimPrefix(line, "/nick "))
					if newNick != "" {
						c.nick = newNick
					} else {
						c.ui.ErrorMessage(errors.New("invalid nickname"))
					}
					return
				}

				if line == "/help" {
					c.ui.InfoMessage(`Available commands:
  /connect multiaddress - dials a node adding it to the list of connected peers
  /peers - list of peers connected to this node
  /nick newNick - change the user's nickname
  /exit - closes the app`)
					return
				}

				c.SendMessage(line)
			}()
		}
	}
}

func (c *Chat) SendMessage(line string) {
	tCtx, cancel := context.WithTimeout(c.ctx, 3*time.Second)
	defer func() {
		cancel()
	}()

	err := c.publish(tCtx, line)
	if err != nil {
		if err.Error() == "validation failed" {
			err = errors.New("message rate violation!")
		}
		c.ui.ErrorMessage(err)
	}
}

func (c *Chat) publish(ctx context.Context, message string) error {
	msg := &pb.Chat2Message{
		Timestamp: uint64(c.node.Timesource().Now().Unix()),
		Nick:      c.nick,
		Payload:   []byte(message),
	}

	msgBytes, err := proto.Marshal(msg)
	if err != nil {
		return err
	}

	var version uint32
	var timestamp int64 = utils.GetUnixEpochFrom(c.node.Timesource().Now())
	var keyInfo *payload.KeyInfo = &payload.KeyInfo{}

	if c.options.UsePayloadV1 { // Use WakuV1 encryption
		keyInfo.Kind = payload.Symmetric
		keyInfo.SymKey = generateSymKey(c.options.ContentTopic)
		version = 1
	} else {
		keyInfo.Kind = payload.None
		version = 0
	}

	p := new(payload.Payload)
	p.Data = msgBytes
	p.Key = keyInfo

	payload, err := p.Encode(version)
	if err != nil {
		return err
	}

	wakuMsg := &wpb.WakuMessage{
		Payload:      payload,
		Version:      version,
		ContentTopic: options.ContentTopic,
		Timestamp:    timestamp,
	}

	if c.options.RLNRelay.Enable {
		// for future version when we support more than one rln protected content topic,
		// we should check the message content topic as well
		err = c.node.RLNRelay().AppendRLNProof(wakuMsg, c.node.Timesource().Now())
		if err != nil {
			return err
		}

		c.ui.InfoMessage(fmt.Sprintf("RLN Epoch: %d", rln.BytesToEpoch(wakuMsg.RateLimitProof.Epoch).Uint64()))
	}

	if c.options.LightPush.Enable {
		var lightOpt lightpush.LightPushOption
		var peerID peer.ID
		peerID, err = options.LightPush.NodePeerID()
		if err != nil {
			lightOpt = lightpush.WithAutomaticPeerSelection()
		} else {
			lightOpt = lightpush.WithPeer(peerID)
		}

		_, err = c.node.Lightpush().Publish(c.ctx, wakuMsg, lightOpt)
	} else {
		_, err = c.node.Relay().Publish(ctx, wakuMsg)
	}

	return err
}

func decodeMessage(useV1Payload bool, contentTopic string, wakumsg *wpb.WakuMessage) (*pb.Chat2Message, error) {
	var keyInfo *payload.KeyInfo = &payload.KeyInfo{}
	if useV1Payload { // Use WakuV1 encryption
		keyInfo.Kind = payload.Symmetric
		keyInfo.SymKey = generateSymKey(contentTopic)
	} else {
		keyInfo.Kind = payload.None
	}

	payload, err := payload.DecodePayload(wakumsg, keyInfo)
	if err != nil {
		return nil, err
	}

	msg := &pb.Chat2Message{}
	if err := proto.Unmarshal(payload.Data, msg); err != nil {
		return nil, err
	}

	return msg, nil
}

func generateSymKey(password string) []byte {
	// AesKeyLength represents the length (in bytes) of an private key
	AESKeyLength := 256 / 8
	return pbkdf2.Key([]byte(password), nil, 65356, AESKeyLength, sha256.New)
}

func (c *Chat) retrieveHistory(connectionWg *sync.WaitGroup) {
	defer c.wg.Done()

	connectionWg.Wait() // Wait until node connection operations are done

	if !c.options.Store.Enable {
		return
	}

	var storeOpt store.HistoryRequestOption
	if c.options.Store.Node == nil {
		c.ui.InfoMessage("No store node configured. Choosing one at random...")
		storeOpt = store.WithAutomaticPeerSelection()
	} else {
		peerID, err := (*c.options.Store.Node).ValueForProtocol(multiaddr.P_P2P)
		if err != nil {
			c.ui.ErrorMessage(err)
			return
		}
		pID, err := peer.Decode(peerID)
		if err != nil {
			c.ui.ErrorMessage(err)
			return
		}
		storeOpt = store.WithPeer(pID)
		c.ui.InfoMessage(fmt.Sprintf("Querying historic messages from %s", peerID))

	}

	tCtx, cancel := context.WithTimeout(c.ctx, 10*time.Second)
	defer cancel()

	q := store.Query{
		ContentTopics: []string{options.ContentTopic},
	}

	response, err := c.node.Store().Query(tCtx, q,
		store.WithAutomaticRequestId(),
		storeOpt,
		store.WithPaging(false, 100))

	if err != nil {
		c.ui.ErrorMessage(fmt.Errorf("could not query storenode: %w", err))
	} else {
		if len(response.Messages) == 0 {
			c.ui.InfoMessage("0 historic messages available")
		} else {
			for _, msg := range response.Messages {
				c.C <- protocol.NewEnvelope(msg, msg.Timestamp, relay.DefaultWakuTopic)
			}
		}
	}
}

func (c *Chat) staticNodes(connectionWg *sync.WaitGroup) {
	defer c.wg.Done()
	defer connectionWg.Done()

	<-c.uiReady // wait until UI is ready

	wg := sync.WaitGroup{}

	wg.Add(len(options.StaticNodes))
	for _, n := range options.StaticNodes {
		go func(addr multiaddr.Multiaddr) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(c.ctx, time.Duration(10)*time.Second)
			defer cancel()

			peerID, err := addr.ValueForProtocol(multiaddr.P_P2P)
			if err != nil {
				c.ui.ErrorMessage(err)
				return
			}

			c.ui.InfoMessage(fmt.Sprintf("Connecting to %s", addr.String()))

			err = c.node.DialPeerWithMultiAddress(ctx, addr)
			if err != nil {
				c.ui.ErrorMessage(err)
			} else {
				c.ui.InfoMessage(fmt.Sprintf("Connected to %s", peerID))
			}
		}(n)
	}

	wg.Wait()

}

func (c *Chat) welcomeMessage() {
	defer c.wg.Done()

	<-c.uiReady // wait until UI is ready

	c.ui.InfoMessage("Welcome, " + c.nick + "!")
	c.ui.InfoMessage("type /help to see available commands \n")

	addrMessage := "Listening on:\n"
	for _, addr := range c.node.ListenAddresses() {
		addrMessage += "  -" + addr.String() + "\n"
	}
	c.ui.InfoMessage(addrMessage)

	if !c.options.RLNRelay.Enable {
		return
	}

	credential, err := c.node.RLNRelay().IdentityCredential()
	if err != nil {
		c.ui.Quit()
		fmt.Println(err.Error())
	}

	idx, err := c.node.RLNRelay().MembershipIndex()
	if err != nil {
		c.ui.Quit()
		fmt.Println(err.Error())
	}

	idTrapdoor := credential.IDTrapdoor
	idNullifier := credential.IDSecretHash
	idSecretHash := credential.IDSecretHash
	idCommitment := credential.IDCommitment

	rlnMessage := "RLN config:\n"
	rlnMessage += fmt.Sprintf("- Your membership index is: %d\n", idx)
	rlnMessage += fmt.Sprintf("- Your rln identity trapdoor is: 0x%s\n", hex.EncodeToString(idTrapdoor[:]))
	rlnMessage += fmt.Sprintf("- Your rln identity nullifier is: 0x%s\n", hex.EncodeToString(idNullifier[:]))
	rlnMessage += fmt.Sprintf("- Your rln identity secret hash is: 0x%s\n", hex.EncodeToString(idSecretHash[:]))
	rlnMessage += fmt.Sprintf("- Your rln identity commitment key is: 0x%s\n", hex.EncodeToString(idCommitment[:]))

	c.ui.InfoMessage(rlnMessage)
}

func (c *Chat) discoverNodes(connectionWg *sync.WaitGroup) {
	defer c.wg.Done()
	defer connectionWg.Done()

	<-c.uiReady // wait until UI is ready

	var dnsDiscoveryUrl string
	if options.Fleet != fleetNone {
		if options.Fleet == fleetTest {
			dnsDiscoveryUrl = "enrtree://AOGECG2SPND25EEFMAJ5WF3KSGJNSGV356DSTL2YVLLZWIV6SAYBM@test.waku.nodes.status.im"
		} else {
			// Connect to prod by default
			dnsDiscoveryUrl = "enrtree://AOGECG2SPND25EEFMAJ5WF3KSGJNSGV356DSTL2YVLLZWIV6SAYBM@prod.waku.nodes.status.im"
		}
	}

	if options.DNSDiscovery.Enable && options.DNSDiscovery.URL != "" {
		dnsDiscoveryUrl = options.DNSDiscovery.URL
	}

	if dnsDiscoveryUrl != "" {
		c.ui.InfoMessage(fmt.Sprintf("attempting DNS discovery with %s", dnsDiscoveryUrl))
		nodes, err := dnsdisc.RetrieveNodes(c.ctx, dnsDiscoveryUrl, dnsdisc.WithNameserver(options.DNSDiscovery.Nameserver))
		if err != nil {
			c.ui.ErrorMessage(errors.New(err.Error()))
		} else {
			var nodeList []multiaddr.Multiaddr
			for _, n := range nodes {
				nodeList = append(nodeList, n.Addresses...)
			}
			c.ui.InfoMessage(fmt.Sprintf("Discovered and connecting to %v ", nodeList))
			wg := sync.WaitGroup{}
			wg.Add(len(nodeList))
			for _, n := range nodeList {
				go func(ctx context.Context, addr multiaddr.Multiaddr) {
					defer wg.Done()

					peerID, err := addr.ValueForProtocol(multiaddr.P_P2P)
					if err != nil {
						c.ui.ErrorMessage(err)
						return
					}

					ctx, cancel := context.WithTimeout(ctx, time.Duration(10)*time.Second)
					defer cancel()
					err = c.node.DialPeerWithMultiAddress(ctx, addr)
					if err != nil {
						c.ui.ErrorMessage(fmt.Errorf("could not connect to %s: %w", peerID, err))
					} else {
						c.ui.InfoMessage(fmt.Sprintf("Connected to %s", peerID))
					}
				}(c.ctx, n)

			}
			wg.Wait()
		}
	}
}
