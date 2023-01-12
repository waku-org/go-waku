package discv5

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math"
	"net"
	"strconv"
	"testing"
	"time"

	gcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"go.uber.org/zap"

	"github.com/libp2p/go-libp2p"
	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/discovery"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
)

func createHost(t *testing.T) (host.Host, int, *ecdsa.PrivateKey) {
	privKey, err := gcrypto.GenerateKey()
	require.NoError(t, err)

	sPrivKey := libp2pcrypto.PrivKey(utils.EcdsaPrivKeyToSecp256k1PrivKey(privKey))

	port, err := tests.FindFreePort(t, "127.0.0.1", 3)
	require.NoError(t, err)

	sourceMultiAddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", port))
	require.NoError(t, err)

	host, err := libp2p.New(
		libp2p.ListenAddrs(sourceMultiAddr),
		libp2p.Identity(sPrivKey),
	)
	require.NoError(t, err)

	return host, port, privKey
}

func newLocalnode(priv *ecdsa.PrivateKey, ipAddr *net.TCPAddr, udpPort int, wakuFlags utils.WakuEnrBitfield, advertiseAddr *net.IP, log *zap.Logger) (*enode.LocalNode, error) {
	db, err := enode.OpenDB("")
	if err != nil {
		return nil, err
	}
	localnode := enode.NewLocalNode(db, priv)
	localnode.SetFallbackUDP(udpPort)
	localnode.Set(enr.WithEntry(utils.WakuENRField, wakuFlags))
	localnode.SetFallbackIP(net.IP{127, 0, 0, 1})
	localnode.SetStaticIP(ipAddr.IP)

	if udpPort > 0 && udpPort <= math.MaxUint16 {
		localnode.Set(enr.UDP(uint16(udpPort))) // lgtm [go/incorrect-integer-conversion]
	} else {
		log.Error("setting udpPort", zap.Int("port", udpPort))
	}

	if ipAddr.Port > 0 && ipAddr.Port <= math.MaxUint16 {
		localnode.Set(enr.TCP(uint16(ipAddr.Port))) // lgtm [go/incorrect-integer-conversion]
	} else {
		log.Error("setting tcpPort", zap.Int("port", ipAddr.Port))
	}

	if advertiseAddr != nil {
		localnode.SetStaticIP(*advertiseAddr)
	}

	return localnode, nil
}

func extractIP(addr multiaddr.Multiaddr) (*net.TCPAddr, error) {
	ipStr, err := addr.ValueForProtocol(multiaddr.P_IP4)
	if err != nil {
		return nil, err
	}

	portStr, err := addr.ValueForProtocol(multiaddr.P_TCP)
	if err != nil {
		return nil, err
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, err
	}
	return &net.TCPAddr{
		IP:   net.ParseIP(ipStr),
		Port: port,
	}, nil
}

func TestDiscV5(t *testing.T) {
	// Host1 <-> Host2 <-> Host3

	// H1
	host1, _, prvKey1 := createHost(t)
	udpPort1, err := tests.FindFreePort(t, "127.0.0.1", 3)
	require.NoError(t, err)
	ip1, _ := extractIP(host1.Addrs()[0])
	l1, err := newLocalnode(prvKey1, ip1, udpPort1, utils.NewWakuEnrBitfield(true, true, true, true), nil, utils.Logger())
	require.NoError(t, err)
	d1, err := NewDiscoveryV5(host1, prvKey1, l1, utils.Logger(), WithUDPPort(udpPort1))
	require.NoError(t, err)

	// H2
	host2, _, prvKey2 := createHost(t)
	ip2, _ := extractIP(host2.Addrs()[0])
	udpPort2, err := tests.FindFreePort(t, "127.0.0.1", 3)
	require.NoError(t, err)
	l2, err := newLocalnode(prvKey2, ip2, udpPort2, utils.NewWakuEnrBitfield(true, true, true, true), nil, utils.Logger())
	require.NoError(t, err)
	d2, err := NewDiscoveryV5(host2, prvKey2, l2, utils.Logger(), WithUDPPort(udpPort2), WithBootnodes([]*enode.Node{d1.localnode.Node()}))
	require.NoError(t, err)

	// H3
	host3, _, prvKey3 := createHost(t)
	ip3, _ := extractIP(host3.Addrs()[0])
	udpPort3, err := tests.FindFreePort(t, "127.0.0.1", 3)
	require.NoError(t, err)
	l3, err := newLocalnode(prvKey3, ip3, udpPort3, utils.NewWakuEnrBitfield(true, true, true, true), nil, utils.Logger())
	require.NoError(t, err)
	d3, err := NewDiscoveryV5(host3, prvKey3, l3, utils.Logger(), WithUDPPort(udpPort3), WithBootnodes([]*enode.Node{d2.localnode.Node()}))
	require.NoError(t, err)

	defer d1.Stop()
	defer d2.Stop()
	defer d3.Stop()

	err = d1.Start(context.Background())
	require.NoError(t, err)

	err = d2.Start(context.Background())
	require.NoError(t, err)

	err = d3.Start(context.Background())
	require.NoError(t, err)

	time.Sleep(3 * time.Second) // Wait for nodes to be discovered

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	d3.Discover(ctx, 2)
	foundHost1 := host3.Network().Connectedness(host1.ID()) == network.Connected
	foundHost2 := host3.Network().Connectedness(host2.ID()) == network.Connected
	require.True(t, foundHost1 && foundHost2)

	// Should return nodes from the cache
	d3.Stop()

	ctx1, cancel1 := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel1()

	results, err := d3.FindNodes(ctx1, "", discovery.Limit(2))
	require.NoError(t, err)
	require.Len(t, results, 2)

	// Simulate empty cache

	for i := range d3.peerCache.recs {
		delete(d3.peerCache.recs, i)
	}

	ctx2, cancel2 := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel2()

	results, err = d3.FindNodes(ctx2, "", discovery.Limit(2))
	require.NoError(t, err)
	require.Len(t, results, 0)

	// Restart peer search
	err = d3.Start(context.Background())
	require.NoError(t, err)

	time.Sleep(3 * time.Second) // Wait for nodes to be discovered

	ctx3, cancel3 := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel3()

	d3.Discover(ctx3, 2)
	foundHost1 = host3.Network().Connectedness(host1.ID()) == network.Connected
	foundHost2 = host3.Network().Connectedness(host2.ID()) == network.Connected

	require.True(t, foundHost1 && foundHost2)

}
