package node

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	proto "github.com/golang/protobuf/proto"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr-net"

	"github.com/status-im/go-waku/waku/v2/protocol"
)

// Default clientId
const clientId string = "Go Waku v2 node"

type Topic string

const DefaultWakuTopic Topic = "/waku/2/default-waku/proto"

type Message []byte

type WakuInfo struct {
	// NOTE One for simplicity, can extend later as needed
	listenStr        string
	multiaddrStrings []byte
}

type MessagePair struct {
	a *Topic
	b *protocol.WakuMessage
}

type Subscription struct {
	C               chan *protocol.WakuMessage
	closed          bool
	mutex           sync.Mutex
	pubSubscription *pubsub.Subscription
	quit            chan struct{}
}

type WakuNode struct {
	// peerManager *PeerManager
	host   host.Host
	pubsub *pubsub.PubSub

	topics     map[Topic]*pubsub.Topic
	topicsLock sync.Mutex

	// peerInfo peer.AddrInfo
	// TODO Revisit messages field indexing as well as if this should be Message or WakuMessage
	// messages []MessagePair

	subscriptions protocol.MessageNotificationSubscriptions
	ctx           context.Context
	cancel        context.CancelFunc
	privKey       crypto.PrivKey
}

func New(ctx context.Context, privKey *ecdsa.PrivateKey, hostAddr net.Addr, extAddr net.Addr) (*WakuNode, error) {
	// Creates a Waku Node.
	if hostAddr == nil {
		return nil, errors.New("Host address cannot be null")
	}

	var multiAddresses []ma.Multiaddr
	hostAddrMA, err := manet.FromNetAddr(hostAddr)
	if err != nil {
		return nil, err
	}
	multiAddresses = append(multiAddresses, hostAddrMA)

	if extAddr != nil {
		extAddrMA, err := manet.FromNetAddr(extAddr)
		if err != nil {
			return nil, err
		}
		multiAddresses = append(multiAddresses, extAddrMA)
	}

	nodeKey := crypto.PrivKey((*crypto.Secp256k1PrivateKey)(privKey))

	opts := []libp2p.Option{
		libp2p.ListenAddrs(multiAddresses...),
		libp2p.Identity(nodeKey),
		libp2p.DefaultTransports, //
		libp2p.NATPortMap(),      // Attempt to open ports using uPNP for NATed hosts.
		//libp2p.DisableRelay(),     // TODO: is this needed?
		//libp2p.EnableNATService(), // TODO: is this needed?
	}

	ctx, cancel := context.WithCancel(ctx)
	_ = cancel

	host, err := libp2p.New(ctx, opts...)
	if err != nil {
		return nil, err
	}

	w := new(WakuNode)
	//w.peerManager = NewPeerManager(host)
	w.pubsub = nil
	w.host = host
	w.cancel = cancel
	w.privKey = nodeKey
	w.ctx = ctx
	w.topics = make(map[Topic]*pubsub.Topic)

	hostInfo, _ := ma.NewMultiaddr(fmt.Sprintf("/p2p/%s", host.ID().Pretty()))
	for _, addr := range host.Addrs() {
		fullAddr := addr.Encapsulate(hostInfo)
		log.Printf("Listening on %s\n", fullAddr)
	}

	return w, nil
}

func (w *WakuNode) Stop() error {
	defer w.cancel()
	// TODO: Is it necessary to stop WakuRelaySubRouter?
	return nil
}

func (w *WakuNode) Host() host.Host {
	return w.host
}

func (w *WakuNode) PubSub() *pubsub.PubSub {
	return w.pubsub
}

func (w *WakuNode) SetPubSub(pubSub *pubsub.PubSub) {
	w.pubsub = pubSub
}

func (w *WakuNode) MountRelay() error {
	ps, err := protocol.NewWakuRelay(w.ctx, w.host)
	if err != nil {
		return err
	}
	w.pubsub = ps

	// TODO: filters
	// TODO: rlnRelay

	return nil
}

func (node *WakuNode) Subscribe(topic *Topic) (*Subscription, error) {
	// Subscribes to a PubSub topic. Triggers handler when receiving messages on
	// this topic. TopicHandler is a method that takes a topic and some data.
	// NOTE The data field SHOULD be decoded as a WakuMessage.

	if node.pubsub == nil {
		return nil, errors.New("PubSub hasn't been set. Execute mountWakuRelay() or setPubSub() first")
	}

	pubSubTopic, err := node.upsertTopic(topic)
	if err != nil {
		return nil, err
	}

	sub, err := pubSubTopic.Subscribe()
	if err != nil {
		return nil, err
	}

	subscription := new(Subscription)
	subscription.closed = false
	subscription.pubSubscription = sub
	subscription.C = make(chan *protocol.WakuMessage)
	subscription.quit = make(chan struct{})

	go func(ctx context.Context, sub *pubsub.Subscription) {
		nextMsgTicker := time.NewTicker(time.Millisecond * 10)
		defer nextMsgTicker.Stop()

		for {
			select {
			case <-subscription.quit:
				subscription.mutex.Lock()
				defer subscription.mutex.Unlock()
				close(subscription.C)
				subscription.closed = true
				return
			case <-nextMsgTicker.C:
				msg, err := sub.Next(ctx)

				if err != nil {
					fmt.Println("Error receiving message", err)
					return // Should close channel?
				}

				wakuMessage := &protocol.WakuMessage{}
				if err := proto.Unmarshal(msg.Data, wakuMessage); err != nil {
					fmt.Println("Error decoding WakuMessage: ", err) // TODO: use log lib
					return
				}

				subscription.C <- wakuMessage
			}
		}
	}(node.ctx, sub)

	return subscription, nil
}

func (subs *Subscription) Unsubscribe() {
	// Unsubscribes a handler from a PubSub topic.
	subs.mutex.Lock()
	defer subs.mutex.Unlock()
	if !subs.closed {
		close(subs.quit)
	}
}

func (subs *Subscription) IsClosed() bool {
	subs.mutex.Lock()
	defer subs.mutex.Unlock()
	return subs.closed
}

func (node *WakuNode) upsertTopic(topic *Topic) (*pubsub.Topic, error) {
	defer node.topicsLock.Unlock()
	node.topicsLock.Lock()

	var t Topic = DefaultWakuTopic
	if topic != nil {
		t = *topic
	}

	pubSubTopic, ok := node.topics[t]
	if !ok { // Joins topic if node hasn't joined yet
		newTopic, err := node.pubsub.Join(string(t))
		if err != nil {
			return nil, err
		}
		node.topics[t] = newTopic
		pubSubTopic = newTopic
	}
	return pubSubTopic, nil
}

func (node *WakuNode) Publish(message *protocol.WakuMessage, topic *Topic) error {
	// Publish a `WakuMessage` to a PubSub topic. `WakuMessage` should contain a
	// `contentTopic` field for light node functionality. This field may be also
	// be omitted.

	if node.pubsub == nil {
		return errors.New("PubSub hasn't been set. Execute mountWakuRelay() or setPubSub() first")
	}

	if message == nil {
		return errors.New("Message can't be null")
	}

	pubSubTopic, err := node.upsertTopic(topic)
	if err != nil {
		return err
	}

	out, err := proto.Marshal(message)
	if err != nil {
		return err
	}

	err = pubSubTopic.Publish(node.ctx, out)
	if err != nil {
		return err
	}

	return nil
}
