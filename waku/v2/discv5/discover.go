package discv5

import (
	"context"
	"crypto/ecdsa"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/status-im/go-waku/waku/v2/utils"
)

var log = logging.Logger("waku_discv5")

type DiscoveryV5 struct {
	discovery.Discovery

	params    *discV5Parameters
	host      host.Host
	config    discover.Config
	udpAddr   *net.UDPAddr
	listener  *discover.UDPv5
	localnode *enode.LocalNode

	peerCache peerCache
}

type peerCache struct {
	sync.RWMutex
	recs map[peer.ID]peerRecord
	rng  *rand.Rand
}

type peerRecord struct {
	expire int64
	peer   peer.AddrInfo
}

type discV5Parameters struct {
	host             host.Host
	bootnodes        []*enode.Node
	advertiseAddress *net.IP
	udpPort          int
}

const WakuENRField = "waku2"

// WakuEnrBitfield is a8-bit flag field to indicate Waku capabilities. Only the 4 LSBs are currently defined according to RFC31 (https://rfc.vac.dev/spec/31/).
type WakuEnrBitfield = uint8

type DiscoveryV5Option func(*discV5Parameters)

func WithBootnodes(bootnodes []*enode.Node) DiscoveryV5Option {
	return func(params *discV5Parameters) {
		params.bootnodes = bootnodes
	}
}

func WithAdvertiseAddress(advertiseAddr net.IP) DiscoveryV5Option {
	return func(params *discV5Parameters) {
		params.advertiseAddress = &advertiseAddr
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

func NewWakuEnrBitfield(lightpush, filter, store, relay bool) WakuEnrBitfield {
	var v uint8 = 0

	if lightpush {
		v |= (1 << 3)
	}

	if filter {
		v |= (1 << 2)
	}

	if store {
		v |= (1 << 1)
	}

	if relay {
		v |= (1 << 0)
	}

	return v
}

func NewDiscoveryV5(host host.Host, ipAddr net.IP, tcpPort int, priv *ecdsa.PrivateKey, wakuFlags WakuEnrBitfield, opts ...DiscoveryV5Option) (*DiscoveryV5, error) {
	params := new(discV5Parameters)
	optList := DefaultOptions()
	optList = append(optList, opts...)
	for _, opt := range optList {
		opt(params)
	}

	localnode, err := newLocalnode(priv, ipAddr, params.udpPort, tcpPort, wakuFlags, params.advertiseAddress)
	if err != nil {
		return nil, err
	}

	discV5 := new(DiscoveryV5)
	discV5.host = host
	discV5.params = params
	discV5.peerCache.rng = rand.New(rand.NewSource(rand.Int63()))
	discV5.peerCache.recs = make(map[peer.ID]peerRecord)
	discV5.localnode = localnode
	discV5.config = discover.Config{
		PrivateKey: priv,
		Bootnodes:  params.bootnodes,
	}
	discV5.udpAddr = &net.UDPAddr{
		IP:   ipAddr,
		Port: params.udpPort,
	}

	return discV5, nil
}

func newLocalnode(priv *ecdsa.PrivateKey, ipAddr net.IP, udpPort int, tcpPort int, wakuFlags WakuEnrBitfield, advertiseAddr *net.IP) (*enode.LocalNode, error) {
	db, err := enode.OpenDB("")
	if err != nil {
		return nil, err
	}

	localnode := enode.NewLocalNode(db, priv)
	localnode.SetFallbackIP(net.IP{127, 0, 0, 1})
	localnode.SetFallbackUDP(udpPort)

	if advertiseAddr != nil {
		localnode.SetStaticIP(*advertiseAddr)
	}

	localnode.Set(enr.WithEntry(WakuENRField, wakuFlags))

	localnode.Set(enr.IP(ipAddr))
	localnode.Set(enr.UDP(udpPort))
	localnode.Set(enr.TCP(tcpPort))

	return localnode, nil
}

func (d *DiscoveryV5) Start() error {
	conn, err := net.ListenUDP("udp", d.udpAddr)
	if err != nil {
		return err
	}

	listener, err := discover.ListenV5(conn, d.localnode, d.config)
	if err != nil {
		return err
	}

	d.listener = listener

	log.Info("Started Discovery V5 at %s:%d", d.udpAddr.IP, d.udpAddr.Port)

	return nil
}

func (d *DiscoveryV5) Stop() {
	d.listener.Close()
}

func isWakuNode(node *enode.Node) bool {
	enrField := new(WakuEnrBitfield)
	if err := node.Record().Load(enr.WithEntry(WakuENRField, &enrField)); err != nil {
		if !enr.IsNotFound(err) {
			log.Error("could not retrieve port for enr ", node)
		}
		return false
	}

	if enrField != nil {
		return *enrField != uint8(0)
	}

	return false
}

func (d *DiscoveryV5) evaluateNode(node *enode.Node) bool {
	if node == nil || node.IP() == nil {
		return false
	}

	enrTCP := new(enr.TCP)
	if err := node.Record().Load(enr.WithEntry(enrTCP.ENRKey(), enrTCP)); err != nil {
		if !enr.IsNotFound(err) {
			log.Error("could not retrieve port for enr ", node)
		}
		return false
	}

	if !isWakuNode(node) {
		return false
	}

	address, err := utils.EnodeToMultiAddr(node)
	if err != nil {
		log.Error("could not convert enode to multiaddress ", err)
		return false
	}

	info, err := peer.AddrInfoFromP2pAddr(address)
	if err != nil {
		log.Error("could not obtain peer info from multiaddress ", err)
		return false
	}

	if d.host.Network().Connectedness(info.ID) == network.Connected {
		return false
	}

	return true
}

func (c *DiscoveryV5) Advertise(ctx context.Context, ns string, opts ...discovery.Option) (time.Duration, error) {
	// Get options
	var options discovery.Options
	err := options.Apply(opts...)
	if err != nil {
		return 0, err
	}

	// TODO: once discv5 spec introduces capability and topic discovery, implement this function

	return 20 * time.Minute, nil
}

func (d *DiscoveryV5) iterate(iterator enode.Iterator, limit int, doneCh chan struct{}) {
	for {
		if len(d.peerCache.recs) >= limit {
			break
		}

		exists := iterator.Next()
		if !exists {
			break
		}

		address, err := utils.EnodeToMultiAddr(iterator.Node())
		if err != nil {
			log.Error(err)
			continue
		}

		peerInfo, err := peer.AddrInfoFromP2pAddr(address)
		if err != nil {
			log.Error(err)
			continue
		}

		d.peerCache.recs[peerInfo.ID] = peerRecord{
			expire: time.Now().Unix() + 3600, // Expires in 1hr
			peer:   *peerInfo,
		}
	}

	close(doneCh)
}

func (d *DiscoveryV5) FindPeers(ctx context.Context, topic string, opts ...discovery.Option) (<-chan peer.AddrInfo, error) {
	// Get options
	var options discovery.Options
	err := options.Apply(opts...)
	if err != nil {
		return nil, err
	}

	const maxLimit = 100
	limit := options.Limit
	if limit == 0 || limit > maxLimit {
		limit = maxLimit
	}

	// We are ignoring the topic. Future versions might use a map[string]*peerCache instead where the string represents the pubsub topic

	d.peerCache.Lock()
	defer d.peerCache.Unlock()

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

	// Discover new records if we don't have enough
	if newCacheSize < limit {
		iterator := d.listener.RandomNodes()
		iterator = enode.Filter(iterator, d.evaluateNode)
		defer iterator.Close()

		doneCh := make(chan struct{})
		go d.iterate(iterator, limit, doneCh)

		select {
		case <-ctx.Done():
		case <-doneCh:
		}
	}

	// Randomize and fill channel with available records
	count := len(d.peerCache.recs)
	if limit < count {
		count = limit
	}

	chPeer := make(chan peer.AddrInfo, count)

	perm := d.peerCache.rng.Perm(len(d.peerCache.recs))[0:count]
	permSet := make(map[int]int)
	for i, v := range perm {
		permSet[v] = i
	}

	sendLst := make([]*peer.AddrInfo, count)
	iter := 0
	for k := range d.peerCache.recs {
		if sendIndex, ok := permSet[iter]; ok {
			peerInfo := d.peerCache.recs[k].peer
			sendLst[sendIndex] = &peerInfo
		}
		iter++
	}

	for _, send := range sendLst {
		chPeer <- *send
	}

	close(chPeer)

	return chPeer, err
}
