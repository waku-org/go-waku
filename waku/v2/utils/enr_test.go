package utils

import (
	"fmt"
	"net"
	"testing"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	gcrypto "github.com/status-im/go-waku/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestEnodeToMultiAddr(t *testing.T) {
	enr := "enr:-IS4QAmC_o1PMi5DbR4Bh4oHVyQunZblg4bTaottPtBodAhJZvxVlWW-4rXITPNg4mwJ8cW__D9FBDc9N4mdhyMqB-EBgmlkgnY0gmlwhIbRi9KJc2VjcDI1NmsxoQOevTdO6jvv3fRruxguKR-3Ge4bcFsLeAIWEDjrfaigNoN0Y3CCdl8"

	parsedNode := enode.MustParse(enr)
	expectedMultiAddr := "/ip4/134.209.139.210/tcp/30303/p2p/16Uiu2HAmPLe7Mzm8TsYUubgCAW1aJoeFScxrLj8ppHFivPo97bUZ"
	actualMultiAddr, err := enodeToMultiAddr(parsedNode)
	require.NoError(t, err)
	require.Equal(t, expectedMultiAddr, actualMultiAddr.String())
}

func TestGetENRandIP(t *testing.T) {
	key, _ := gcrypto.GenerateKey()
	pubKey := EcdsaPubKeyToSecp256k1PublicKey(&key.PublicKey)
	id, _ := peer.IDFromPublicKey(pubKey)

	hostAddr := &net.TCPAddr{IP: net.ParseIP("192.168.0.1"), Port: 9999}
	hostMultiAddr, _ := manet.FromNetAddr(hostAddr)
	hostInfo, _ := ma.NewMultiaddr(fmt.Sprintf("/p2p/%s", id.Pretty()))
	ogMultiaddress := hostMultiAddr.Encapsulate(hostInfo)

	wakuFlag := NewWakuEnrBitfield(true, true, true, true)

	node, resTCPAddr, err := GetENRandIP(ogMultiaddress, wakuFlag, key)
	require.NoError(t, err)
	require.Equal(t, hostAddr, resTCPAddr)

	parsedNode := enode.MustParse(node.String())
	resMultiaddress, err := enodeToMultiAddr(parsedNode)
	require.NoError(t, err)
	require.Equal(t, ogMultiaddress.String(), resMultiaddress.String())
}

func TestMultiaddr(t *testing.T) {
	key, _ := gcrypto.GenerateKey()
	pubKey := EcdsaPubKeyToSecp256k1PublicKey(&key.PublicKey)
	id, _ := peer.IDFromPublicKey(pubKey)
	ogMultiaddress, _ := ma.NewMultiaddr("/ip4/10.0.0.241/tcp/60001/ws/p2p/" + id.Pretty())
	wakuFlag := NewWakuEnrBitfield(true, true, true, true)

	node, _, err := GetENRandIP(ogMultiaddress, wakuFlag, key)
	require.NoError(t, err)

	multiaddresses, err := Multiaddress(node)
	require.NoError(t, err)
	require.Len(t, multiaddresses, 1)
	require.True(t, ogMultiaddress.Equal(multiaddresses[0]))
}
