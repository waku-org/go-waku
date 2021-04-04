package node

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	proto "github.com/golang/protobuf/proto"
	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr-net"
	common "github.com/status-im/go-waku/waku/common"
	"github.com/status-im/go-waku/waku/v2/protocol"
	store "github.com/status-im/go-waku/waku/v2/protocol/waku_store"
	wakurelay "github.com/status-im/go-wakurelay-pubsub"
)

var log = logging.Logger("wakunode")

// Default clientId
const clientId string = "Go Waku v2 node"

type Topic string

const DefaultWakuTopic Topic = "/waku/2/default-waku/proto"

type Message []byte

type Subscription struct {
	C               chan *common.Envelope
	closed          bool
	mutex           sync.Mutex
	pubSubscription *wakurelay.Subscription
	quit            chan struct{}
}

type WakuNode struct {
	host   host.Host
	pubsub *wakurelay.PubSub
	store  *store.WakuStore

	topics      map[Topic]*wakurelay.Topic
	topicsMutex sync.Mutex

	subscriptions      []*Subscription
	subscriptionsMutex sync.Mutex

	ctx     context.Context
	cancel  context.CancelFunc
	privKey crypto.PrivKey
}

func New(ctx context.Context, privKey *ecdsa.PrivateKey, hostAddr []net.Addr, opts ...libp2p.Option) (*WakuNode, error) {
	// Creates a Waku Node.
	if hostAddr == nil {
		return nil, errors.New("host address cannot be null")
	}

	var multiAddresses []ma.Multiaddr
	for _, addr := range hostAddr {
		hostAddrMA, err := manet.FromNetAddr(addr)
		if err != nil {
			return nil, err
		}
		multiAddresses = append(multiAddresses, hostAddrMA)
	}

	nodeKey := crypto.PrivKey((*crypto.Secp256k1PrivateKey)(privKey))

	opts = append(opts,
		libp2p.ListenAddrs(multiAddresses...),
		libp2p.Identity(nodeKey),
		libp2p.DefaultTransports,
		libp2p.NATPortMap(),                                           // Attempt to open ports using uPNP for NATed hosts.
		libp2p.EnableNATService(),                                     // TODO: is this needed?)
		libp2p.ConnectionManager(connmgr.NewConnManager(200, 300, 0)), // ?
	)

	ctx, cancel := context.WithCancel(ctx)
	_ = cancel

	host, err := libp2p.New(ctx, opts...)
	if err != nil {
		return nil, err
	}

	w := new(WakuNode)
	w.pubsub = nil
	w.host = host
	w.cancel = cancel
	w.privKey = nodeKey
	w.ctx = ctx
	w.topics = make(map[Topic]*wakurelay.Topic)

	for _, addr := range w.ListenAddresses() {
		log.Info("Listening on", addr)
	}

	return w, nil
}

func (w *WakuNode) Stop() {
	w.subscriptionsMutex.Lock()
	defer w.subscriptionsMutex.Unlock()
	defer w.cancel()

	for _, sub := range w.subscriptions {
		sub.Unsubscribe()
	}

	w.subscriptions = nil
}

func (w *WakuNode) Host() host.Host {
	return w.host
}

func (w *WakuNode) ID() string {
	return w.host.ID().Pretty()
}

func (w *WakuNode) ListenAddresses() []string {
	hostInfo, _ := ma.NewMultiaddr(fmt.Sprintf("/p2p/%s", w.host.ID().Pretty()))
	var result []string
	for _, addr := range w.host.Addrs() {
		result = append(result, addr.Encapsulate(hostInfo).String())
	}
	return result
}

func (w *WakuNode) PubSub() *wakurelay.PubSub {
	return w.pubsub
}

func (w *WakuNode) SetPubSub(pubSub *wakurelay.PubSub) {
	w.pubsub = pubSub
}

func (w *WakuNode) MountRelay(opts ...wakurelay.Option) error {
	ps, err := wakurelay.NewWakuRelaySub(w.ctx, w.host, opts...)
	if err != nil {
		return err
	}
	w.pubsub = ps

	// TODO: filters
	// TODO: rlnRelay

	return nil
}

func (w *WakuNode) MountStore(s store.MessageProvider) error {
	sub, err := w.Subscribe(nil)
	if err != nil {
		return err
	}
	w.store = store.NewWakuStore(w.ctx, w.host, sub.C, s)
	return nil
}

func (w *WakuNode) StartStore() error {
	if w.store == nil {
		return errors.New("WakuStore is not set")
	}

	w.store.Start()
	return nil
}

func (w *WakuNode) AddStorePeer(address string) error {
	if w.store == nil {
		return errors.New("WakuStore is not set")
	}

	storePeer, err := ma.NewMultiaddr(address)
	if err != nil {
		return err
	}

	// Extract the peer ID from the multiaddr.
	info, err := peer.AddrInfoFromP2pAddr(storePeer)
	if err != nil {
		return err
	}

	return w.store.AddPeer(info.ID, info.Addrs)
}

func (w *WakuNode) Query(contentTopic uint32, asc bool, pageSize int64) (*protocol.HistoryResponse, error) {
	if w.store == nil {
		return nil, errors.New("WakuStore is not set")
	}

	query := new(protocol.HistoryQuery)
	query.Topics = append(query.Topics, contentTopic)
	query.PagingInfo = new(protocol.PagingInfo)
	if asc {
		query.PagingInfo.Direction = protocol.PagingInfo_FORWARD
	} else {
		query.PagingInfo.Direction = protocol.PagingInfo_BACKWARD
	}
	query.PagingInfo.PageSize = pageSize

	result, err := w.store.Query(query)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (node *WakuNode) Subscribe(topic *Topic) (*Subscription, error) {
	// Subscribes to a PubSub topic.
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
	subscription.C = make(chan *common.Envelope)
	subscription.quit = make(chan struct{})

	go func(ctx context.Context, sub *wakurelay.Subscription) {
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
					subscription.mutex.Lock()
					defer subscription.mutex.Unlock()
					if !subscription.closed {
						subscription.closed = true
						close(subscription.quit)
					}
					return
				}

				wakuMessage := &protocol.WakuMessage{}
				if err := proto.Unmarshal(msg.Data, wakuMessage); err != nil {
					log.Error("could not decode message", err)
					return
				}

				envelope := common.NewEnvelope(wakuMessage, len(msg.Data), sha256.Sum256(msg.Data))
				subscription.C <- envelope
			}
		}
	}(node.ctx, sub)

	node.subscriptionsMutex.Lock()
	defer node.subscriptionsMutex.Unlock()

	node.subscriptions = append(node.subscriptions, subscription)

	return subscription, nil
}

func (subs *Subscription) Unsubscribe() {
	// Unsubscribes a handler from a PubSub topic.
	subs.mutex.Lock()
	defer subs.mutex.Unlock()
	if !subs.closed {
		subs.closed = true
		close(subs.quit)
	}
}

func (subs *Subscription) IsClosed() bool {
	subs.mutex.Lock()
	defer subs.mutex.Unlock()
	return subs.closed
}

func (node *WakuNode) upsertTopic(topic *Topic) (*wakurelay.Topic, error) {
	defer node.topicsMutex.Unlock()
	node.topicsMutex.Lock()

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
		return errors.New("message can't be null")
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

func (w *WakuNode) DialPeer(address string) error {
	p, err := ma.NewMultiaddr(address)
	if err != nil {
		return err
	}

	// Extract the peer ID from the multiaddr.
	info, err := peer.AddrInfoFromP2pAddr(p)
	if err != nil {
		return err
	}

	w.host.Connect(w.ctx, *info)
	return nil
}

func (w *WakuNode) ClosePeerByAddress(address string) error {
	p, err := ma.NewMultiaddr(address)
	if err != nil {
		return err
	}

	// Extract the peer ID from the multiaddr.
	info, err := peer.AddrInfoFromP2pAddr(p)
	if err != nil {
		return err
	}

	return w.ClosePeerById(info.ID)
}

func (w *WakuNode) ClosePeerById(id peer.ID) error {
	return w.host.Network().ClosePeer(id)
}

func (w *WakuNode) PeerCount() int {
	return len(w.host.Network().Peers())
}
