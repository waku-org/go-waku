package discv5

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"net"
	"testing"
	"time"

	gcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/multiformats/go-multiaddr"
	"github.com/status-im/go-waku/tests"
	"github.com/stretchr/testify/require"

	"github.com/libp2p/go-libp2p"
	libp2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/host"
)

func createHost(t *testing.T) (host.Host, int, *ecdsa.PrivateKey) {
	privKey, err := gcrypto.GenerateKey()
	require.NoError(t, err)

	sPrivKey := libp2pcrypto.PrivKey((*libp2pcrypto.Secp256k1PrivateKey)(privKey))

	port, err := tests.FindFreePort(t, "127.0.0.1", 3)
	require.NoError(t, err)

	sourceMultiAddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", port))
	require.NoError(t, err)

	host, err := libp2p.New(
		context.Background(),
		libp2p.ListenAddrs(sourceMultiAddr),
		libp2p.Identity(sPrivKey),
	)
	require.NoError(t, err)

	return host, port, privKey
}

func TestDiscV5(t *testing.T) {
	// Host1 <-> Host2 <-> Host3

	host1, tcpPort1, prvKey1 := createHost(t)
	udpPort1, err := tests.FindFreePort(t, "127.0.0.1", 3)
	require.NoError(t, err)
	d1, err := NewDiscoveryV5(host1, net.IPv4(127, 0, 0, 1), tcpPort1, prvKey1, NewWakuEnrBitfield(true, true, true, true), WithUDPPort(udpPort1))
	require.NoError(t, err)

	host2, tcpPort2, prvKey2 := createHost(t)
	udpPort2, err := tests.FindFreePort(t, "127.0.0.1", 3)
	require.NoError(t, err)
	d2, err := NewDiscoveryV5(host2, net.IPv4(127, 0, 0, 1), tcpPort2, prvKey2, NewWakuEnrBitfield(true, true, true, true), WithUDPPort(udpPort2), WithBootnodes([]*enode.Node{d1.localnode.Node()}))
	require.NoError(t, err)

	host3, tcpPort3, prvKey3 := createHost(t)
	udpPort3, err := tests.FindFreePort(t, "127.0.0.1", 3)
	require.NoError(t, err)
	d3, err := NewDiscoveryV5(host3, net.IPv4(127, 0, 0, 1), tcpPort3, prvKey3, NewWakuEnrBitfield(true, true, true, true), WithUDPPort(udpPort3), WithBootnodes([]*enode.Node{d2.localnode.Node()}))
	require.NoError(t, err)

	err = d1.Start()
	require.NoError(t, err)

	err = d2.Start()
	require.NoError(t, err)

	err = d3.Start()
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	peerChan, err := d1.FindPeers(ctx, "", discovery.Limit(2))
	require.NoError(t, err)

	foundHost2 := false
	foundHost3 := false
	for p := range peerChan {
		fmt.Println(p)
		if p.Addrs[0].String() == host2.Addrs()[0].String() {
			foundHost2 = true
		}

		if p.Addrs[0].String() == host3.Addrs()[0].String() {
			foundHost3 = true

		}
	}

	require.True(t, foundHost2 && foundHost3)
}
