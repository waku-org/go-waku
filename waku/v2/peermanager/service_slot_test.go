package peermanager

import (
	"testing"

	"github.com/libp2p/go-libp2p/core/peer"
	libp2pProtocol "github.com/libp2p/go-libp2p/core/protocol"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/maps"
)

func TestServiceSlotRemovePeerFromAll(t *testing.T) {
	slots := NewServiceSlot()

	protocol := libp2pProtocol.ID("/vac/waku/test/2.0.0")
	protocol1 := libp2pProtocol.ID("/vac/waku/test/2.0.2")

	peerID := peer.ID("peerId")

	//
	slots.getPeers(protocol).add(peerID)
	slots.getPeers(protocol1).add(peerID)
	//
	fetchedPeers, err := slots.getPeers(protocol1).getRandom(1)
	require.NoError(t, err)
	require.Equal(t, peerID, maps.Keys(fetchedPeers)[0])

	//
	slots.removePeer(peerID)
	//
	_, err = slots.getPeers(protocol).getRandom(1)
	require.Equal(t, err, ErrNoPeersAvailable)
	_, err = slots.getPeers(protocol1).getRandom(1)
	require.Equal(t, err, ErrNoPeersAvailable)
}
