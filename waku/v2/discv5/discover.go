package discv5

import (
	"context"
	"crypto/ecdsa"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/discovery"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/backoff"
	"github.com/waku-org/go-discover/discover"
	"github.com/waku-org/go-waku/logging"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/ethereum/go-ethereum/p2p/nat"
)

const dialTimeout = 30 * time.Second

type DiscoveryV5 struct {
	sync.RWMutex

	discovery.Discovery

	params    *discV5Parameters
	host      host.Host
	config    discover.Config
	udpAddr   *net.UDPAddr
	listener  *discover.UDPv5
	localnode *enode.LocalNode
	NAT       nat.Interface
	connector *backoff.BackoffConnector

	log *zap.Logger

	started bool
	cancel  context.CancelFunc
	wg      *sync.WaitGroup

	peerCache peerCache
}

type peerCache struct {
	sync.RWMutex
	recs map[peer.ID]PeerRecord
	rng  *rand.Rand
}

type PeerRecord struct {
	expire int64
	Peer   peer.AddrInfo
	Node   *enode.Node
}

type discV5Parameters struct {
	autoUpdate    bool
	bootnodes     []*enode.Node
	udpPort       int
	advertiseAddr *net.IP
}

type DiscoveryV5Option func(*discV5Parameters)

var protocolID = [6]byte{'d', '5', 'w', 'a', 'k', 'u'}

func WithAutoUpdate(autoUpdate bool) DiscoveryV5Option {
	return func(params *discV5Parameters) {
		params.autoUpdate = autoUpdate
	}
}

func WithBootnodes(bootnodes []*enode.Node) DiscoveryV5Option {
	return func(params *discV5Parameters) {
		params.bootnodes = bootnodes
	}
}

func WithAdvertiseAddr(addr net.IP) DiscoveryV5Option {
	return func(params *discV5Parameters) {
		params.advertiseAddr = &addr
	}
}

func WithUDPPort(port int) DiscoveryV5Option {
	return func(params *discV5Parameters) {
		params.udpPort = port
	}
}

func DefaultOptions() []DiscoveryV5Option {
	return []DiscoveryV5Option{
		WithUDPPort(9000),
	}
}

const MaxPeersToDiscover = 600

func NewDiscoveryV5(host host.Host, priv *ecdsa.PrivateKey, localnode *enode.LocalNode, log *zap.Logger, opts ...DiscoveryV5Option) (*DiscoveryV5, error) {
	params := new(discV5Parameters)
	optList := DefaultOptions()
	optList = append(optList, opts...)
	for _, opt := range optList {
		opt(params)
	}

	logger := log.Named("discv5")

	var NAT nat.Interface = nil
	if params.advertiseAddr == nil {
		NAT = nat.Any()
	}

	cacheSize := 600
	rngSrc := rand.NewSource(rand.Int63())
	minBackoff, maxBackoff := time.Second*30, time.Hour
	bkf := backoff.NewExponentialBackoff(minBackoff, maxBackoff, backoff.FullJitter, time.Second, 5.0, 0, rand.New(rngSrc))
	connector, err := backoff.NewBackoffConnector(host, cacheSize, dialTimeout, bkf)
	if err != nil {
		return nil, err
	}

	return &DiscoveryV5{
		host:      host,
		connector: connector,
		params:    params,
		NAT:       NAT,
		wg:        &sync.WaitGroup{},
		peerCache: peerCache{
			rng:  rand.New(rand.NewSource(rand.Int63())),
			recs: make(map[peer.ID]PeerRecord),
		},
		localnode: localnode,
		config: discover.Config{
			PrivateKey: priv,
			Bootnodes:  params.bootnodes,
			V5Config: discover.V5Config{
				ProtocolID: &protocolID,
			},
		},
		udpAddr: &net.UDPAddr{
			IP:   net.IPv4zero,
			Port: params.udpPort,
		},
		log: logger,
	}, nil
}

func (d *DiscoveryV5) Node() *enode.Node {
	return d.localnode.Node()
}

func (d *DiscoveryV5) listen(ctx context.Context) error {
	conn, err := net.ListenUDP("udp", d.udpAddr)
	if err != nil {
		return err
	}

	d.udpAddr = conn.LocalAddr().(*net.UDPAddr)
	if d.NAT != nil && !d.udpAddr.IP.IsLoopback() {
		d.wg.Add(1)
		go func() {
			defer d.wg.Done()
			nat.Map(d.NAT, ctx.Done(), "udp", d.udpAddr.Port, d.udpAddr.Port, "go-waku discv5 discovery")
		}()

	}

	d.localnode.SetFallbackUDP(d.udpAddr.Port)

	listener, err := discover.ListenV5(conn, d.localnode, d.config)
	if err != nil {
		return err
	}

	d.listener = listener

	d.log.Info("started Discovery V5",
		zap.Stringer("listening", d.udpAddr),
		logging.TCPAddr("advertising", d.localnode.Node().IP(), d.localnode.Node().TCP()))
	d.log.Info("Discovery V5: discoverable ENR ", logging.ENode("enr", d.localnode.Node()))

	return nil
}

func (d *DiscoveryV5) Start(ctx context.Context) error {
	d.Lock()
	defer d.Unlock()

	d.wg.Wait() // Waiting for any go routines to stop
	ctx, cancel := context.WithCancel(ctx)

	d.cancel = cancel
	d.started = true

	err := d.listen(ctx)
	if err != nil {
		return err
	}

	d.wg.Add(1)
	go d.runDiscoveryV5Loop(ctx)

	return nil
}

func (d *DiscoveryV5) SetBootnodes(nodes []*enode.Node) error {
	return d.listener.SetFallbackNodes(nodes)
}

func (d *DiscoveryV5) Stop() {
	d.Lock()
	defer d.Unlock()

	if d.cancel == nil {
		return
	}

	d.cancel()
	d.started = false

	if d.listener != nil {
		d.listener.Close()
		d.listener = nil
		d.log.Info("stopped Discovery V5")
	}

	d.wg.Wait()
}

/*
func isWakuNode(node *enode.Node) bool {
	enrField := new(utils.WakuEnrBitfield)
	if err := node.Record().Load(enr.WithEntry(utils.WakuENRField, &enrField)); err != nil {
		if !enr.IsNotFound(err) {
			utils.Logger().Named("discv5").Error("could not retrieve port for enr ", zap.Any("node", node))
		}
		return false
	}

	if enrField != nil {
		return *enrField != uint8(0)
	}

	return false
}
*/

func hasTCPPort(node *enode.Node) bool {
	enrTCP := new(enr.TCP)
	if err := node.Record().Load(enr.WithEntry(enrTCP.ENRKey(), enrTCP)); err != nil {
		if !enr.IsNotFound(err) {
			utils.Logger().Named("discv5").Error("retrieving port for enr", logging.ENode("enr", node))
		}
		return false
	}

	return true
}

func evaluateNode(node *enode.Node) bool {
	if node == nil || node.IP() == nil {
		return false
	}

	//  TODO: consider node filtering based on ENR; we do not filter based on ENR in the first waku discv5 beta stage
	if /*!isWakuNode(node) ||*/ !hasTCPPort(node) {
		return false
	}

	_, err := utils.EnodeToPeerInfo(node)

	if err != nil {
		utils.Logger().Named("discv5").Error("obtaining peer info from enode", logging.ENode("enr", node), zap.Error(err))
		return false
	}

	return true
}

func (d *DiscoveryV5) iterate(ctx context.Context, wg *sync.WaitGroup, iterator enode.Iterator, limit int, outOfPeers chan struct{}) {
	defer wg.Done()

	for {
		if len(d.peerCache.recs) >= limit {
			time.Sleep(1 * time.Minute)
		}

		if ctx.Err() != nil {
			break
		}

		exists := iterator.Next()
		if !exists {
			outOfPeers <- struct{}{}
			break
		}

		addresses, err := utils.Multiaddress(iterator.Node())
		if err != nil {
			d.log.Error("extracting multiaddrs from enr", zap.Error(err))
			continue
		}

		peerAddrs, err := peer.AddrInfosFromP2pAddrs(addresses...)
		if err != nil {
			d.log.Error("converting multiaddrs to addrinfos", zap.Error(err))
			continue
		}

		d.peerCache.Lock()
		for _, p := range peerAddrs {
			_, ok := d.peerCache.recs[p.ID]
			if ok {
				continue
			}

			d.peerCache.recs[p.ID] = PeerRecord{
				expire: time.Now().Unix() + 3600, // Expires in 1hr
				Peer:   p,
				Node:   iterator.Node(),
			}
		}
		d.peerCache.Unlock()
	}
}

func (d *DiscoveryV5) removeExpiredPeers() int {
	// Remove all expired entries from cache
	currentTime := time.Now().Unix()
	newCacheSize := len(d.peerCache.recs)

	for p := range d.peerCache.recs {
		rec := d.peerCache.recs[p]
		if rec.expire < currentTime {
			newCacheSize--
			delete(d.peerCache.recs, p)
		}
	}

	return newCacheSize
}

func (d *DiscoveryV5) runDiscoveryV5Loop(ctx context.Context) {
	defer d.wg.Done()

	ch := make(chan struct{}, 1)
	ch <- struct{}{} // Initial execution

	var iterator enode.Iterator = nil

	iteratorWg := sync.WaitGroup{}

restartLoop:
	for {
		select {
		case <-ch:
			if iterator != nil {
				iterator.Close()
			}
			if d.listener == nil {
				break
			}
			iterator := d.listener.RandomNodes()
			iterator = enode.Filter(iterator, evaluateNode)
			iteratorWg.Add(1)
			go d.iterate(ctx, &iteratorWg, iterator, MaxPeersToDiscover, ch)
		case <-ctx.Done():
			iteratorWg.Wait()
			close(ch)
			break restartLoop
		}
	}
	d.log.Warn("Discv5 loop stopped")
}

func (d *DiscoveryV5) FindNodes(ctx context.Context, topic string, opts ...discovery.Option) ([]PeerRecord, error) {
	// Get options
	var options discovery.Options
	err := options.Apply(opts...)
	if err != nil {
		return nil, err
	}

	limit := options.Limit
	if limit == 0 || limit > MaxPeersToDiscover {
		limit = MaxPeersToDiscover
	}

	// We are ignoring the topic. Future versions might use a map[string]*peerCache instead where the string represents the pubsub topic

	d.peerCache.Lock()
	defer d.peerCache.Unlock()

	d.removeExpiredPeers()

	// Randomize and fill channel with available records
	count := len(d.peerCache.recs)
	if limit < count {
		count = limit
	}

	perm := d.peerCache.rng.Perm(len(d.peerCache.recs))[0:count]
	permSet := make(map[int]int)
	for i, v := range perm {
		permSet[v] = i
	}

	sendLst := make([]PeerRecord, count)
	iter := 0
	for k := range d.peerCache.recs {
		if sendIndex, ok := permSet[iter]; ok {
			sendLst[sendIndex] = d.peerCache.recs[k]
		}
		iter++
	}

	return sendLst, err
}

func (d *DiscoveryV5) Discover(ctx context.Context, numPeers int) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			currPeers := len(d.host.Network().Peers())
			if currPeers < numPeers {
				d.peerCache.Lock()
				d.removeExpiredPeers()

				peersToConnect := numPeers - currPeers

				// Randomize and fill channel with available records
				count := len(d.peerCache.recs)
				if peersToConnect < count {
					count = peersToConnect
				}

				perm := d.peerCache.rng.Perm(len(d.peerCache.recs))[0:count]
				permSet := make(map[int]int)
				for i, v := range perm {
					permSet[v] = i
				}

				chPeer := make(chan peer.AddrInfo, count)
				iter := 0
				for k, p := range d.peerCache.recs {
					if _, ok := permSet[iter]; ok {
						if d.host.Network().Connectedness(k) == network.NotConnected {
							chPeer <- d.peerCache.recs[k].Peer
							p.expire -= 900 // Make the peers expire earlier (reduce 15m)
							d.peerCache.recs[k] = p
						}
					}
					iter++
				}

				d.connector.Connect(ctx, chPeer)

				defer d.peerCache.Unlock()
			}
		}
	}
}

func (d *DiscoveryV5) IsStarted() bool {
	d.RLock()
	defer d.RUnlock()

	return d.started
}
