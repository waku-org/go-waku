package node

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	gcrypto "github.com/ethereum/go-ethereum/crypto"
	proto "github.com/golang/protobuf/proto"
	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
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

type WakuNode struct {
	host   host.Host
	opts   *WakuNodeParameters
	pubsub *wakurelay.PubSub

	topics          map[Topic]bool
	topicsMutex     sync.Mutex
	wakuRelayTopics map[Topic]*wakurelay.Topic

	subscriptions      map[Topic][]*Subscription
	subscriptionsMutex sync.Mutex

	bcaster   Broadcaster
	relaySubs map[Topic]*wakurelay.Subscription

	ctx    context.Context
	cancel context.CancelFunc
}

func New(ctx context.Context, opts ...WakuNodeOption) (*WakuNode, error) {
	params := new(WakuNodeParameters)

	ctx, cancel := context.WithCancel(ctx)
	_ = cancel

	params.ctx = ctx
	params.libP2POpts = DefaultLibP2POptions

	for _, opt := range opts {
		err := opt(params)
		if err != nil {
			return nil, err
		}
	}

	if len(params.multiAddr) > 0 {
		params.libP2POpts = append(params.libP2POpts, libp2p.ListenAddrs(params.multiAddr...))
	}

	if params.privKey != nil {
		params.libP2POpts = append(params.libP2POpts, libp2p.Identity(*params.privKey))
	}

	host, err := libp2p.New(ctx, params.libP2POpts...)
	if err != nil {
		return nil, err
	}

	w := new(WakuNode)
	w.bcaster = NewBroadcaster(1024)
	w.pubsub = nil
	w.host = host
	w.cancel = cancel
	w.ctx = ctx
	w.topics = make(map[Topic]bool)
	w.wakuRelayTopics = make(map[Topic]*wakurelay.Topic)
	w.relaySubs = make(map[Topic]*wakurelay.Subscription)
	w.subscriptions = make(map[Topic][]*Subscription)
	w.opts = params

	if params.enableRelay {
		err := w.mountRelay(params.wOpts...)
		if err != nil {
			return nil, err
		}
	}

	if params.enableStore {
		err := w.startStore()
		if err != nil {
			return nil, err
		}
	}

	for _, addr := range w.ListenAddresses() {
		log.Info("Listening on ", addr)
	}

	return w, nil
}

func (w *WakuNode) Stop() {
	w.subscriptionsMutex.Lock()
	defer w.subscriptionsMutex.Unlock()
	defer w.cancel()

	for topic, _ := range w.topics {
		for _, sub := range w.subscriptions[topic] {
			sub.Unsubscribe()
		}
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

func (w *WakuNode) mountRelay(opts ...wakurelay.Option) error {
	ps, err := wakurelay.NewWakuRelaySub(w.ctx, w.host, opts...)
	if err != nil {
		return err
	}
	w.pubsub = ps

	// TODO: filters
	// TODO: rlnRelay

	log.Info("Relay protocol started")

	return nil
}

func (w *WakuNode) startStore() error {
	_, err := w.Subscribe(nil)
	if err != nil {
		return err
	}

	w.opts.store.Start(w.host)
	return nil
}

func (w *WakuNode) AddStorePeer(address string) (*peer.ID, error) {
	if w.opts.store == nil {
		return nil, errors.New("WakuStore is not set")
	}

	storePeer, err := ma.NewMultiaddr(address)
	if err != nil {
		return nil, err
	}

	// Extract the peer ID from the multiaddr.
	info, err := peer.AddrInfoFromP2pAddr(storePeer)
	if err != nil {
		return nil, err
	}

	return &info.ID, w.opts.store.AddPeer(info.ID, info.Addrs)
}

func (w *WakuNode) Query(contentTopics []string, startTime float64, endTime float64, opts ...store.HistoryRequestOption) (*protocol.HistoryResponse, error) {
	if w.opts.store == nil {
		return nil, errors.New("WakuStore is not set")
	}

	query := new(protocol.HistoryQuery)

	for _, ct := range contentTopics {
		query.ContentFilters = append(query.ContentFilters, &protocol.ContentFilter{ContentTopic: ct})
	}

	query.StartTime = startTime
	query.EndTime = endTime
	query.PagingInfo = new(protocol.PagingInfo)
	result, err := w.opts.store.Query(query, opts...)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func getTopic(topic *Topic) Topic {
	var t Topic = DefaultWakuTopic
	if topic != nil {
		t = *topic
	}
	return t
}

func (node *WakuNode) Subscribe(topic *Topic) (*Subscription, error) {
	// Subscribes to a PubSub topic.
	// NOTE The data field SHOULD be decoded as a WakuMessage.
	if node.pubsub == nil {
		return nil, errors.New("PubSub hasn't been set. Execute mountWakuRelay() or setPubSub() first")
	}

	t := getTopic(topic)

	sub, err := node.upsertSubscription(t)
	if err != nil {
		return nil, err
	}

	// Create client subscription
	subscription := new(Subscription)
	subscription.closed = false
	subscription.C = make(chan *common.Envelope, 1024) // To avoid blocking
	subscription.quit = make(chan struct{})

	node.subscriptionsMutex.Lock()
	defer node.subscriptionsMutex.Unlock()
	node.subscriptions[t] = append(node.subscriptions[t], subscription)

	node.bcaster.Register(subscription.C)

	go func() {
		nextMsgTicker := time.NewTicker(time.Millisecond * 10)
		defer nextMsgTicker.Stop()

		for {
			select {
			case <-subscription.quit:
				subscription.mutex.Lock()
				node.bcaster.Unregister(subscription.C) // Remove from broadcast list
				close(subscription.C)
				subscription.mutex.Unlock()
			case <-nextMsgTicker.C:
				msg, err := sub.Next(node.ctx)
				if err != nil {
					subscription.mutex.Lock()
					node.topicsMutex.Lock()
					for _, subscription := range node.subscriptions[t] {
						subscription.Unsubscribe()
					}
					node.topicsMutex.Unlock()
					subscription.mutex.Unlock()
					return
				}

				wakuMessage := &protocol.WakuMessage{}
				if err := proto.Unmarshal(msg.Data, wakuMessage); err != nil {
					log.Error("could not decode message", err)
					return
				}

				envelope := common.NewEnvelope(wakuMessage, len(msg.Data), gcrypto.Keccak256(msg.Data))

				node.bcaster.Submit(envelope)
			}
		}
	}()

	return subscription, nil
}

func (node *WakuNode) upsertTopic(topic Topic) (*wakurelay.Topic, error) {
	defer node.topicsMutex.Unlock()
	node.topicsMutex.Lock()

	node.topics[topic] = true
	pubSubTopic, ok := node.wakuRelayTopics[topic]
	if !ok { // Joins topic if node hasn't joined yet
		newTopic, err := node.pubsub.Join(string(topic))
		if err != nil {
			return nil, err
		}
		node.wakuRelayTopics[topic] = newTopic
		pubSubTopic = newTopic
	}
	return pubSubTopic, nil
}

func (node *WakuNode) upsertSubscription(topic Topic) (*wakurelay.Subscription, error) {
	sub, ok := node.relaySubs[topic]
	if !ok {
		pubSubTopic, err := node.upsertTopic(topic)
		if err != nil {
			return nil, err
		}

		sub, err = pubSubTopic.Subscribe()
		if err != nil {
			return nil, err
		}
		node.relaySubs[topic] = sub

		log.Info("Subscribing to topic ", topic)

		if node.opts.store != nil && node.opts.storeMsgs {
			log.Info("Subscribing store to topic ", topic)
			node.bcaster.Register(node.opts.store.MsgC)
		}
	}

	return sub, nil
}

func (node *WakuNode) Publish(message *protocol.WakuMessage, topic *Topic) ([]byte, error) {
	// Publish a `WakuMessage` to a PubSub topic. `WakuMessage` should contain a
	// `contentTopic` field for light node functionality. This field may be also
	// be omitted.

	if node.pubsub == nil {
		return nil, errors.New("PubSub hasn't been set. Execute mountWakuRelay() or setPubSub() first")
	}

	if message == nil {
		return nil, errors.New("message can't be null")
	}

	pubSubTopic, err := node.upsertTopic(getTopic(topic))

	if err != nil {
		return nil, err
	}

	out, err := proto.Marshal(message)
	if err != nil {
		return nil, err
	}

	err = pubSubTopic.Publish(node.ctx, out)

	if err != nil {
		return nil, err
	}

	hash := gcrypto.Keccak256(out)

	return hash, nil
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
