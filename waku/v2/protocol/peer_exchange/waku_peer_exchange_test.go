package peer_exchange

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
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/discv5"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"go.uber.org/zap"

	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
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

func TestRetrieveProvidePeerExchangePeers(t *testing.T) {
	// H1
	host1, _, prvKey1 := createHost(t)
	udpPort1, err := tests.FindFreePort(t, "127.0.0.1", 3)
	require.NoError(t, err)
	ip1, _ := extractIP(host1.Addrs()[0])
	l1, err := newLocalnode(prvKey1, ip1, udpPort1, utils.NewWakuEnrBitfield(false, false, false, true), nil, utils.Logger())
	require.NoError(t, err)
	discv5PeerConn1 := tests.NewTestPeerDiscoverer()
	d1, err := discv5.NewDiscoveryV5(prvKey1, l1, discv5PeerConn1, utils.Logger(), discv5.WithUDPPort(uint(udpPort1)))
	require.NoError(t, err)
	d1.SetHost(host1)

	// H2
	host2, _, prvKey2 := createHost(t)
	ip2, _ := extractIP(host2.Addrs()[0])
	udpPort2, err := tests.FindFreePort(t, "127.0.0.1", 3)
	require.NoError(t, err)
	l2, err := newLocalnode(prvKey2, ip2, udpPort2, utils.NewWakuEnrBitfield(false, false, false, true), nil, utils.Logger())
	require.NoError(t, err)
	discv5PeerConn2 := tests.NewTestPeerDiscoverer()
	d2, err := discv5.NewDiscoveryV5(prvKey2, l2, discv5PeerConn2, utils.Logger(), discv5.WithUDPPort(uint(udpPort2)), discv5.WithBootnodes([]*enode.Node{d1.Node()}))
	require.NoError(t, err)
	d2.SetHost(host2)

	// H3
	host3, _, _ := createHost(t)

	defer d1.Stop()
	defer d2.Stop()
	defer host1.Close()
	defer host2.Close()
	defer host3.Close()

	err = d1.Start(context.Background())
	require.NoError(t, err)

	err = d2.Start(context.Background())
	require.NoError(t, err)

	time.Sleep(3 * time.Second) // Wait some time for peers to be discovered

	// mount peer exchange
	pxPeerConn1 := tests.NewTestPeerDiscoverer()
	px1, err := NewWakuPeerExchange(d1, pxPeerConn1, utils.Logger())
	require.NoError(t, err)
	px1.SetHost(host1)

	pxPeerConn3 := tests.NewTestPeerDiscoverer()
	px3, err := NewWakuPeerExchange(nil, pxPeerConn3, utils.Logger())
	require.NoError(t, err)
	px3.SetHost(host3)

	err = px1.Start(context.Background())
	require.NoError(t, err)

	err = px3.Start(context.Background())
	require.NoError(t, err)

	host3.Peerstore().AddAddrs(host1.ID(), host1.Addrs(), peerstore.PermanentAddrTTL)
	err = host3.Peerstore().AddProtocols(host1.ID(), PeerExchangeID_v20alpha1)
	require.NoError(t, err)

	err = px3.Request(context.Background(), 1)
	require.NoError(t, err)

	time.Sleep(3 * time.Second) //  Give the algorithm some time to work its magic

	require.True(t, pxPeerConn3.HasPeer(host2.ID()))

	px1.Stop()
	px3.Stop()
}
