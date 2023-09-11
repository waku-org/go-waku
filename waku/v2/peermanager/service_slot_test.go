package peermanager

import (
	"testing"

	"github.com/libp2p/go-libp2p/core/peer"
	libp2pProtocol "github.com/libp2p/go-libp2p/core/protocol"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

func TestServiceSlot(t *testing.T) {
	slots := NewServiceSlot()

	protocol := libp2pProtocol.ID("test/protocol")

	peerID := peer.ID("peerId")

	//
	slots.getPeers(protocol).add(peerID)
	//
	fetchedPeer, err := slots.getPeers(protocol).getRandom()
	require.NoError(t, err)
	require.Equal(t, peerID, fetchedPeer)

	//
	slots.getPeers(protocol).remove(peerID)
	//
	_, err = slots.getPeers(protocol).getRandom()
	require.Equal(t, err, utils.ErrNoPeersAvailable)
}

func TestServiceSlotRemovePeerFromAll(t *testing.T) {
	slots := NewServiceSlot()

	protocol := libp2pProtocol.ID("test/protocol")
	protocol1 := libp2pProtocol.ID("test/protocol1")

	peerID := peer.ID("peerId")

	//
	slots.getPeers(protocol).add(peerID)
	slots.getPeers(protocol1).add(peerID)
	//
	fetchedPeer, err := slots.getPeers(protocol1).getRandom()
	require.NoError(t, err)
	require.Equal(t, peerID, fetchedPeer)

	//
	slots.removePeer(peerID)
	//
	_, err = slots.getPeers(protocol).getRandom()
	require.Equal(t, err, utils.ErrNoPeersAvailable)
	_, err = slots.getPeers(protocol1).getRandom()
	require.Equal(t, err, utils.ErrNoPeersAvailable)
}
