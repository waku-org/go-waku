package utils

import (
	"context"
	"crypto/rand"
	"fmt"
	"net"
	"testing"
	"time"

	gcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enode"
	crypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/status-im/go-waku/tests"
	"github.com/stretchr/testify/require"
)

func TestSelectPeer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	h1, err := tests.MakeHost(ctx, 0, rand.Reader)
	require.NoError(t, err)
	defer h1.Close()

	h2, err := tests.MakeHost(ctx, 0, rand.Reader)
	require.NoError(t, err)
	defer h2.Close()

	h3, err := tests.MakeHost(ctx, 0, rand.Reader)
	require.NoError(t, err)
	defer h3.Close()

	proto := "test/protocol"

	h1.Peerstore().AddAddrs(h2.ID(), h2.Network().ListenAddresses(), peerstore.PermanentAddrTTL)
	h1.Peerstore().AddAddrs(h3.ID(), h2.Network().ListenAddresses(), peerstore.PermanentAddrTTL)

	// No peers with selected protocol
	_, err = SelectPeer(h1, proto)
	require.Error(t, ErrNoPeersAvailable, err)

	// Peers with selected protocol
	_ = h1.Peerstore().AddProtocols(h2.ID(), proto)
	_ = h1.Peerstore().AddProtocols(h3.ID(), proto)

	_, err = SelectPeerWithLowestRTT(ctx, h1, proto)
	require.NoError(t, err)

}

func TestSelectPeerWithLowestRTT(t *testing.T) {
	// help-wanted: how to slowdown the ping response to properly test the rtt

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	h1, err := tests.MakeHost(ctx, 0, rand.Reader)
	require.NoError(t, err)
	defer h1.Close()

	h2, err := tests.MakeHost(ctx, 0, rand.Reader)
	require.NoError(t, err)
	defer h2.Close()

	h3, err := tests.MakeHost(ctx, 0, rand.Reader)
	require.NoError(t, err)
	defer h3.Close()

	proto := "test/protocol"

	h1.Peerstore().AddAddrs(h2.ID(), h2.Network().ListenAddresses(), peerstore.PermanentAddrTTL)
	h1.Peerstore().AddAddrs(h3.ID(), h2.Network().ListenAddresses(), peerstore.PermanentAddrTTL)

	// No peers with selected protocol
	_, err = SelectPeerWithLowestRTT(ctx, h1, proto)
	require.Error(t, ErrNoPeersAvailable, err)

	// Peers with selected protocol
	_ = h1.Peerstore().AddProtocols(h2.ID(), proto)
	_ = h1.Peerstore().AddProtocols(h3.ID(), proto)

	_, err = SelectPeerWithLowestRTT(ctx, h1, proto)
	require.NoError(t, err)
}

func TestEnodeToMultiAddr(t *testing.T) {
	enr := "enr:-IS4QAmC_o1PMi5DbR4Bh4oHVyQunZblg4bTaottPtBodAhJZvxVlWW-4rXITPNg4mwJ8cW__D9FBDc9N4mdhyMqB-EBgmlkgnY0gmlwhIbRi9KJc2VjcDI1NmsxoQOevTdO6jvv3fRruxguKR-3Ge4bcFsLeAIWEDjrfaigNoN0Y3CCdl8"

	parsedNode := enode.MustParse(enr)
	expectedMultiAddr := "/ip4/134.209.139.210/tcp/30303/p2p/16Uiu2HAmPLe7Mzm8TsYUubgCAW1aJoeFScxrLj8ppHFivPo97bUZ"
	actualMultiAddr, err := EnodeToMultiAddr(parsedNode)
	require.NoError(t, err)
	require.Equal(t, expectedMultiAddr, actualMultiAddr.String())
}

func TestGetENRandIP(t *testing.T) {
	key, _ := gcrypto.GenerateKey()
	privKey := crypto.PrivKey((*crypto.Secp256k1PrivateKey)(key))
	id, _ := peer.IDFromPublicKey(privKey.GetPublic())

	hostAddr := &net.TCPAddr{IP: net.ParseIP("192.168.0.1"), Port: 9999}
	hostMultiAddr, _ := manet.FromNetAddr(hostAddr)
	hostInfo, _ := ma.NewMultiaddr(fmt.Sprintf("/p2p/%s", id.Pretty()))
	ogMultiaddress := hostMultiAddr.Encapsulate(hostInfo)

	node, resTCPAddr, err := GetENRandIP(ogMultiaddress, key)
	require.NoError(t, err)
	require.Equal(t, hostAddr, resTCPAddr)

	parsedNode := enode.MustParse(node.String())
	resMultiaddress, err := EnodeToMultiAddr(parsedNode)
	require.NoError(t, err)
	require.Equal(t, ogMultiaddress.String(), resMultiaddress.String())
}
