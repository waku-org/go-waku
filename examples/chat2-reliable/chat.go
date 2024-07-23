package main

import (
	"chat2-reliable/pb"
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/bits-and-blooms/bloom/v3"
	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/waku-org/go-waku/waku/v2/dnsdisc"
	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-waku/waku/v2/payload"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/filter"
	"github.com/waku-org/go-waku/waku/v2/protocol/lightpush"
	wpb "github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	wrln "github.com/waku-org/go-waku/waku/v2/protocol/rln"
	"github.com/waku-org/go-waku/waku/v2/protocol/store"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"google.golang.org/protobuf/proto"
)

const (
	maxMessageHistory   = 100
	bloomFilterSize     = 10000
	bloomFilterFPRate   = 0.01
	bufferSweepInterval = 5 * time.Second
	syncMessageInterval = 30 * time.Second
	messageAckTimeout   = 10 * time.Second
)

type Chat struct {
	ctx       context.Context
	wg        sync.WaitGroup
	node      *node.WakuNode
	ui        UI
	uiReady   chan struct{}
	inputChan chan string
	options   Options

	C chan *protocol.Envelope

	nick                 string
	lamportTimestamp     int32
	bloomFilter          *bloom.BloomFilter
	outgoingBuffer       []*pb.Message
	incomingBuffer       []*pb.Message
	messageHistory       []*pb.Message
	receivedBloomFilters map[string]*bloom.BloomFilter
	mutex                sync.Mutex
}

func NewChat(ctx context.Context, node *node.WakuNode, connNotifier <-chan node.PeerConnection, options Options) *Chat {
	chat := &Chat{
		ctx:                  ctx,
		node:                 node,
		options:              options,
		nick:                 options.Nickname,
		uiReady:              make(chan struct{}, 1),
		inputChan:            make(chan string, 100),
		lamportTimestamp:     0,
		bloomFilter:          bloom.NewWithEstimates(bloomFilterSize, bloomFilterFPRate),
		outgoingBuffer:       make([]*pb.Message, 0),
		incomingBuffer:       make([]*pb.Message, 0),
		messageHistory:       make([]*pb.Message, 0),
		receivedBloomFilters: make(map[string]*bloom.BloomFilter),
	}

	chat.ui = NewUIModel(chat.uiReady, chat.inputChan)

	topics := options.Relay.Topics.Value()
	if len(topics) == 0 {
		topics = append(topics, relay.DefaultWakuTopic)
	}

	if options.Filter.Enable {
		cf := protocol.ContentFilter{
			PubsubTopic:   relay.DefaultWakuTopic,
			ContentTopics: protocol.NewContentTopicSet(options.ContentTopic),
		}
		var filterOpt filter.FilterSubscribeOption
		peerID, err := options.Filter.NodePeerID()
		if err != nil {
			filterOpt = filter.WithAutomaticPeerSelection()
		} else {
			filterOpt = filter.WithPeer(peerID)
			chat.ui.InfoMessage(fmt.Sprintf("Subscribing to filter node %s", peerID))
		}
		theFilters, err := node.FilterLightnode().Subscribe(ctx, cf, filterOpt)
		if err != nil {
			chat.ui.ErrorMessage(err)
		} else {
			chat.C = theFilters[0].C //Picking first subscription since there is only 1 contentTopic specified.
		}
	} else {
		for _, topic := range topics {
			sub, err := node.Relay().Subscribe(ctx, protocol.NewContentFilter(topic))
			if err != nil {
				chat.ui.ErrorMessage(err)
			} else {
				chat.C = make(chan *protocol.Envelope)
				go func() {
					for e := range sub[0].Ch {
						chat.C <- e
					}
				}()
			}
		}
	}

	connWg := sync.WaitGroup{}
	connWg.Add(2)

	chat.wg.Add(9) // Added 2 more goroutines for periodic tasks
	go chat.parseInput()
	go chat.receiveMessages()
	go chat.welcomeMessage()
	go chat.connectionWatcher(connNotifier)
	go chat.staticNodes(&connWg)
	go chat.discoverNodes(&connWg)
	go chat.retrieveHistory(&connWg)
	go chat.periodicBufferSweep()
	go chat.periodicSyncMessage()

	return chat
}

func (c *Chat) Stop() {
	c.wg.Wait()
	close(c.inputChan)
}

func (c *Chat) connectionWatcher(connNotifier <-chan node.PeerConnection) {
	defer c.wg.Done()
	for {
		select {
		case conn := <-connNotifier:
			if conn.Connected {
				c.ui.InfoMessage(fmt.Sprintf("Peer %s connected", conn.PeerID.String()))
			} else {
				c.ui.InfoMessage(fmt.Sprintf("Peer %s disconnected", conn.PeerID.String()))
			}
		case <-c.ctx.Done():
			return
		}
	}
}

func (c *Chat) receiveMessages() {
	defer c.wg.Done()
	for {
		select {
		case <-c.ctx.Done():
			return
		case value := <-c.C:
			msgContentTopic := value.Message().ContentTopic
			if msgContentTopic != c.options.ContentTopic {
				continue // Discard messages from other topics
			}

			msg, err := decodeMessage(c.options.ContentTopic, value.Message())
			if err == nil {
				c.processReceivedMessage(msg)
			}
		}
	}
}

func (c *Chat) processReceivedMessage(msg *pb.Message) {
	// Update Lamport timestamp
	c.updateLamportTimestamp(msg.LamportTimestamp)

	c.incLamportTimestamp()

	// Check causal dependencies
	if c.checkCausalDependencies(msg) {
		if msg.Content != "" {
			// Process the message
			c.ui.ChatMessage(int64(c.lamportTimestamp), msg.SenderId, msg.Content)
		}

		// Update bloom filter
		c.updateBloomFilter(msg.MessageId)

		// Add to message history
		c.addToMessageHistory(msg)

		// Update received bloom filter for the sender
		if msg.BloomFilter != nil {
			receivedFilter := &bloom.BloomFilter{}
			err := receivedFilter.UnmarshalBinary(msg.BloomFilter)
			if err == nil {
				c.updateReceivedBloomFilter(msg.SenderId, receivedFilter)
			}
		}
	} else {
		// Request missing dependencies
		for _, depID := range msg.CausalHistory {
			if !c.bloomFilter.Test([]byte(depID)) {
				c.requestMissingMessage(depID)
			}
		}
		// Add to incoming buffer
		c.addIncomingBuffer(msg)
	}
}

func (c *Chat) requestMissingMessage(messageID string) {
	// Implement logic to request a missing message from Store nodes or other participants
	go func() {
		ctx, cancel := context.WithTimeout(c.ctx, 10*time.Second)
		defer cancel()

		hash, err := base64.URLEncoding.DecodeString(messageID)
		if err != nil {
			c.ui.ErrorMessage(fmt.Errorf("failed to parse message hash: %w", err))
			return
		}

		x := store.MessageHashCriteria{
			MessageHashes: []wpb.MessageHash{wpb.ToMessageHash(hash)},
		}

		pID, err := c.getStoreNodePID()
		if err != nil {
			c.ui.ErrorMessage(fmt.Errorf("failed to parse store node peerID: %w", err))
			return
		}
		response, err := c.node.Store().Request(ctx, x,
			store.WithAutomaticRequestID(),
			store.WithPeer(*pID),
			//store.WithAutomaticPeerSelection(),
			store.WithPaging(true, 100), // Use paging to handle potentially large result sets
		)

		if err != nil {
			c.ui.ErrorMessage(fmt.Errorf("failed to retrieve missing message: %w", err))
			return
		}

		// Filter the response to find the specific message
		for _, msg := range response.Messages() {
			decodedMsg, err := decodeMessage(c.options.ContentTopic, msg.Message)
			if err != nil {
				continue
			}
			if decodedMsg.MessageId == messageID {
				c.C <- protocol.NewEnvelope(msg.Message, msg.Message.GetTimestamp(), relay.DefaultWakuTopic)
				return
			}
		}

		c.ui.ErrorMessage(fmt.Errorf("missing message not found: %s", messageID))
	}()
}

func (c *Chat) checkCausalDependencies(msg *pb.Message) bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	for _, depID := range msg.CausalHistory {
		if !c.bloomFilter.Test([]byte(depID)) {
			return false
		}
	}
	return true
}

func (c *Chat) updateBloomFilter(messageID string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.bloomFilter.Add([]byte(messageID))
}

func (c *Chat) addToMessageHistory(msg *pb.Message) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.messageHistory = append(c.messageHistory, msg)
	if len(c.messageHistory) > maxMessageHistory {
		c.messageHistory = c.messageHistory[1:]
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
							peerInfoMsg += fmt.Sprintf("• %s:\n", p.String())

							var strProtocols []string
							for _, p := range peerProtocols {
								strProtocols = append(strProtocols, string(p))
							}

							peerInfoMsg += fmt.Sprintf("    Protocols: %s\n", strings.Join(strProtocols, ", "))
							peerInfoMsg += "    Addresses:\n"
							for _, addr := range peerInfo.Addrs {
								peerInfoMsg += fmt.Sprintf("    - %s/p2p/%s\n", addr.String(), p.String())
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
	c.incLamportTimestamp()

	bloomBytes, err := c.bloomFilterBytes()
	if err != nil {
		c.ui.ErrorMessage(fmt.Errorf("failed to marshal bloom filter: %w", err))
		return
	}

	msg := &pb.Message{
		SenderId:         c.node.Host().ID().String(),
		MessageId:        generateUniqueID(),
		LamportTimestamp: c.lamportTimestamp,
		CausalHistory:    c.getRecentMessageIDs(2),
		ChannelId:        c.options.ContentTopic,
		BloomFilter:      bloomBytes,
		Content:          line,
	}

	c.outgoingBuffer = append(c.outgoingBuffer, msg)

	tCtx, cancel := context.WithTimeout(c.ctx, 3*time.Second)
	defer cancel()

	err = c.publish(tCtx, msg)
	if err != nil {
		if err.Error() == "validation failed" {
			err = errors.New("message rate violation")
		}
		c.ui.ErrorMessage(err)
	} else {
		c.addToMessageHistory(msg)
		c.updateBloomFilter(msg.MessageId)
	}
}

func (c *Chat) publish(ctx context.Context, message *pb.Message) error {
	msgBytes, err := proto.Marshal(message)
	if err != nil {
		return err
	}

	version := uint32(0)
	timestamp := utils.GetUnixEpochFrom(c.node.Timesource().Now())
	keyInfo := &payload.KeyInfo{
		Kind: payload.None,
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
		Version:      proto.Uint32(version),
		ContentTopic: c.options.ContentTopic,
		Timestamp:    timestamp,
	}

	if c.options.RLNRelay.Enable {
		err = c.node.RLNRelay().AppendRLNProof(wakuMsg, c.node.Timesource().Now())
		if err != nil {
			return err
		}

		rateLimitProof, err := wrln.BytesToRateLimitProof(wakuMsg.RateLimitProof)
		if err != nil {
			return err
		}

		c.ui.InfoMessage(fmt.Sprintf("RLN Epoch: %d", rateLimitProof.Epoch.Uint64()))
	}

	if c.options.LightPush.Enable {
		lightOpt := []lightpush.RequestOption{lightpush.WithDefaultPubsubTopic()}
		var peerID peer.ID
		peerID, err = c.options.LightPush.NodePeerID()
		if err != nil {
			lightOpt = append(lightOpt, lightpush.WithAutomaticPeerSelection())
		} else {
			lightOpt = append(lightOpt, lightpush.WithPeer(peerID))
		}

		_, err = c.node.Lightpush().Publish(c.ctx, wakuMsg, lightOpt...)
	} else {
		_, err = c.node.Relay().Publish(ctx, wakuMsg, relay.WithDefaultPubsubTopic())
	}

	return err
}

func decodeMessage(contentTopic string, wakumsg *wpb.WakuMessage) (*pb.Message, error) {
	keyInfo := &payload.KeyInfo{
		Kind: payload.None,
	}

	payload, err := payload.DecodePayload(wakumsg, keyInfo)
	if err != nil {
		return nil, err
	}

	msg := &pb.Message{}
	if err := proto.Unmarshal(payload.Data, msg); err != nil {
		return nil, err
	}

	return msg, nil
}

func (c *Chat) retrieveHistory(connectionWg *sync.WaitGroup) {
	defer c.wg.Done()

	connectionWg.Wait() // Wait until node connection operations are done

	if !c.options.Store.Enable {
		return
	}

	var storeOpt store.RequestOption
	if c.options.Store.Node == nil {
		c.ui.InfoMessage("No store node configured. Choosing one at random...")
		storeOpt = store.WithAutomaticPeerSelection()
	} else {
		pID, err := c.getStoreNodePID()
		if err != nil {
			c.ui.ErrorMessage(err)
			return
		}
		storeOpt = store.WithPeer(*pID)
		c.ui.InfoMessage(fmt.Sprintf("Querying historic messages from %s", pID.String()))
	}

	tCtx, cancel := context.WithTimeout(c.ctx, 10*time.Second)
	defer cancel()

	q := store.FilterCriteria{
		ContentFilter: protocol.NewContentFilter(relay.DefaultWakuTopic, c.options.ContentTopic),
	}

	response, err := c.node.Store().Request(tCtx, q,
		store.WithAutomaticRequestID(),
		storeOpt,
		store.WithPaging(false, 100))

	if err != nil {
		c.ui.ErrorMessage(fmt.Errorf("could not query storenode: %w", err))
	} else {
		if len(response.Messages()) == 0 {
			c.ui.InfoMessage("0 historic messages available")
		} else {
			for _, msg := range response.Messages() {
				c.C <- protocol.NewEnvelope(msg.Message, msg.Message.GetTimestamp(), relay.DefaultWakuTopic)
			}
		}
	}
}

func (c *Chat) staticNodes(connectionWg *sync.WaitGroup) {
	defer c.wg.Done()
	defer connectionWg.Done()

	<-c.uiReady // wait until UI is ready

	wg := sync.WaitGroup{}

	wg.Add(len(c.options.StaticNodes))
	for _, n := range c.options.StaticNodes {
		go func(addr multiaddr.Multiaddr) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(c.ctx, time.Duration(10)*time.Second)
			defer cancel()

			c.ui.InfoMessage(fmt.Sprintf("Connecting to %s", addr.String()))

			err := c.node.DialPeerWithMultiAddress(ctx, addr)
			if err != nil {
				c.ui.ErrorMessage(err)
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
	}

	idx := c.node.RLNRelay().MembershipIndex()

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
	if c.options.DNSDiscovery.Enable {
		if c.options.Fleet != fleetNone {
			if c.options.Fleet == fleetTest {
				dnsDiscoveryUrl = "enrtree://AOGYWMBYOUIMOENHXCHILPKY3ZRFEULMFI4DOM442QSZ73TT2A7VI@test.waku.nodes.status.im"
			} else {
				// Connect to prod by default
				dnsDiscoveryUrl = "enrtree://AIRVQ5DDA4FFWLRBCHJWUWOO6X6S4ZTZ5B667LQ6AJU6PEYDLRD5O@sandbox.waku.nodes.status.im"
			}
		}

		if c.options.DNSDiscovery.URL != "" {
			dnsDiscoveryUrl = c.options.DNSDiscovery.URL
		}
	}

	if dnsDiscoveryUrl != "" {
		c.ui.InfoMessage(fmt.Sprintf("attempting DNS discovery with %s", dnsDiscoveryUrl))
		nodes, err := dnsdisc.RetrieveNodes(c.ctx, dnsDiscoveryUrl, dnsdisc.WithNameserver(c.options.DNSDiscovery.Nameserver))
		if err != nil {
			c.ui.ErrorMessage(errors.New(err.Error()))
		} else {
			var nodeList []peer.AddrInfo
			for _, n := range nodes {
				nodeList = append(nodeList, n.PeerInfo)
			}
			c.ui.InfoMessage(fmt.Sprintf("Discovered and connecting to %v ", nodeList))
			wg := sync.WaitGroup{}
			wg.Add(len(nodeList))
			for _, n := range nodeList {
				go func(ctx context.Context, info peer.AddrInfo) {
					defer wg.Done()

					ctx, cancel := context.WithTimeout(ctx, time.Duration(20)*time.Second)
					defer cancel()
					err = c.node.DialPeerWithInfo(ctx, info)
					if err != nil {
						c.ui.ErrorMessage(fmt.Errorf("could not connect to %s: %w", info.ID.String(), err))
					}
				}(c.ctx, n)
			}
			wg.Wait()
		}
	}
}

func generateUniqueID() string {
	return uuid.New().String()
}

func (c *Chat) getRecentMessageIDs(n int) []string {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	result := make([]string, 0, n)
	for i := len(c.messageHistory) - 1; i >= 0 && len(result) < n; i-- {
		result = append(result, c.messageHistory[i].MessageId)
	}
	return result
}

func (c *Chat) periodicBufferSweep() {
	defer c.wg.Done()

	ticker := time.NewTicker(bufferSweepInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.sweepBuffers()
		}
	}
}

func (c *Chat) getStoreNodePID() (*peer.ID, error) {
	peerID, err := (*c.options.Store.Node).ValueForProtocol(multiaddr.P_P2P)
	if err != nil {
		return nil, err
	}
	pID, err := peer.Decode(peerID)
	if err != nil {
		return nil, err
	}

	return &pID, nil
}

func (c *Chat) sweepBuffers() {
	// Process incoming buffer
	newIncomingBuffer := make([]*pb.Message, 0)
	for _, msg := range c.incomingBuffer {
		if c.checkCausalDependencies(msg) {
			c.processReceivedMessage(msg)
		} else {
			newIncomingBuffer = append(newIncomingBuffer, msg)
		}
	}
	c.setIncomingBuffer(newIncomingBuffer)

	// Resend unacknowledged messages from outgoing buffer
	for _, msg := range c.outgoingBuffer {
		if !c.isMessageAcknowledged(msg) {
			c.resendMessage(msg)
		}
	}
}

func (c *Chat) isMessageAcknowledged(msg *pb.Message) bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	ackCount := 0
	totalPeers := len(c.receivedBloomFilters)

	for _, filter := range c.receivedBloomFilters {
		if filter.Test([]byte(msg.MessageId)) {
			ackCount++
		}
	}

	// Consider a message acknowledged if at least 2/3 of peers have it in their bloom filter
	return ackCount >= (2 * totalPeers / 3)
}

func (c *Chat) resendMessage(msg *pb.Message) {
	// Implement logic to resend the message
	go func() {
		ctx, cancel := context.WithTimeout(c.ctx, messageAckTimeout)
		defer cancel()

		err := c.publish(ctx, msg)
		if err != nil {
			c.ui.ErrorMessage(fmt.Errorf("failed to resend message: %w", err))
		}
	}()
}

func (c *Chat) periodicSyncMessage() {
	defer c.wg.Done()

	ticker := time.NewTicker(syncMessageInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.sendSyncMessage()
		}
	}
}

func (c *Chat) sendSyncMessage() {
	bloomBytes, err := c.bloomFilterBytes()
	if err != nil {
		c.ui.ErrorMessage(fmt.Errorf("failed to marshal bloom filter: %w", err))
		return
	}

	syncMsg := &pb.Message{
		SenderId:         c.node.Host().ID().String(),
		MessageId:        generateUniqueID(),
		LamportTimestamp: c.lamportTimestamp,
		CausalHistory:    c.getRecentMessageIDs(2),
		ChannelId:        c.options.ContentTopic,
		BloomFilter:      bloomBytes,
		Content:          "", // Empty content for sync messages
	}

	err = c.publish(c.ctx, syncMsg)
	if err != nil {
		c.ui.ErrorMessage(fmt.Errorf("failed to send sync message: %w", err))
	}
}

func (c *Chat) addIncomingBuffer(msg *pb.Message) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.incomingBuffer = append(c.incomingBuffer, msg)
}

func (c *Chat) setIncomingBuffer(newBuffer []*pb.Message) {
	c.incomingBuffer = newBuffer
}

func (c *Chat) bloomFilterBytes() ([]byte, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.bloomFilter.MarshalBinary()
}

func (c *Chat) updateReceivedBloomFilter(senderId string, receivedFilter *bloom.BloomFilter) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.receivedBloomFilters[senderId] = receivedFilter
}

func (c *Chat) incLamportTimestamp() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.lamportTimestamp++
}

func (c *Chat) updateLamportTimestamp(msgTs int32) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if msgTs > c.lamportTimestamp {
		c.lamportTimestamp = msgTs
	}
}

func (c *Chat) getLamportTimestamp() int32 {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.lamportTimestamp
}
