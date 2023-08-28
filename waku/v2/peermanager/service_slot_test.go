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

	peerId := peer.ID("peerId")

	//
	slots.getPeers(protocol).add(peerId)
	//
	fetchedPeer, err := slots.getPeers(protocol).getRandom()
	require.NoError(t, err)
	require.Equal(t, peerId, fetchedPeer)

	//
	slots.getPeers(protocol).remove(peerId)
	//
	_, err = slots.getPeers(protocol).getRandom()
	require.Equal(t, err, utils.ErrNoPeersAvailable)
}

func TestServiceSlotRemovePeerFromAll(t *testing.T) {
	slots := NewServiceSlot()

	protocol := libp2pProtocol.ID("test/protocol")
	protocol1 := libp2pProtocol.ID("test/protocol1")

	peerId := peer.ID("peerId")

	//
	slots.getPeers(protocol).add(peerId)
	slots.getPeers(protocol1).add(peerId)
	//
	fetchedPeer, err := slots.getPeers(protocol1).getRandom()
	require.NoError(t, err)
	require.Equal(t, peerId, fetchedPeer)

	//
	slots.RemovePeer(peerId)
	//
	_, err = slots.getPeers(protocol).getRandom()
	require.Equal(t, err, utils.ErrNoPeersAvailable)
	_, err = slots.getPeers(protocol1).getRandom()
	require.Equal(t, err, utils.ErrNoPeersAvailable)
}
