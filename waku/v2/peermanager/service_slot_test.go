package peermanager

import (
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	libp2pProtocol "github.com/libp2p/go-libp2p/core/protocol"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/maps"
)

func TestServiceSlot(t *testing.T) {
	slots := NewServiceSlot()

	protocol := libp2pProtocol.ID("/vac/waku/test/2.0.0")

	peerID := peer.ID("peerId")

	//
	slots.getPeers(protocol).add(peerID)
	//
	fetchedPeers, err := slots.getPeers(protocol).getRandom(1)
	require.NoError(t, err)
	require.Equal(t, peerID, maps.Keys(fetchedPeers)[0])
	//
	slots.getPeers(protocol).remove(peerID)
	//
	_, err = slots.getPeers(protocol).getRandom(1)
	require.Equal(t, err, ErrNoPeersAvailable)

	// Test with more peers
	peerID2 := peer.ID("peerId2")
	peerID3 := peer.ID("peerId3")

	//
	slots.getPeers(protocol).add(peerID2)
	slots.getPeers(protocol).add(peerID3)
	//

	fetchedPeers, err = slots.getPeers(protocol).getRandom(2)
	require.NoError(t, err)
	require.Equal(t, 2, len(maps.Keys(fetchedPeers)))

	time.Sleep(100 * time.Millisecond)

	// Check for uniqueness
	require.NotEqual(t, maps.Keys(fetchedPeers)[0], maps.Keys(fetchedPeers)[1])

	slots.getPeers(protocol).remove(peerID2)

	fetchedPeers, err = slots.getPeers(protocol).getRandom(10)
	require.NoError(t, err)
	require.Equal(t, peerID3, maps.Keys(fetchedPeers)[0])

}

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
