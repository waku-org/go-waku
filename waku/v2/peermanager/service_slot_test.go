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
	slots.GetPeers(protocol).Add(peerId)
	//
	fetchedPeer, err := slots.GetPeers(protocol).GetRandom()
	require.NoError(t, err)
	require.Equal(t, peerId, fetchedPeer)

	//
	slots.GetPeers(protocol).Remove(peerId)
	//
	_, err = slots.GetPeers(protocol).GetRandom()
	require.Equal(t, err, utils.ErrNoPeersAvailable)
}

func TestServiceSlotRemovePeerFromAll(t *testing.T) {
	slots := NewServiceSlot()

	protocol := libp2pProtocol.ID("test/protocol")
	protocol1 := libp2pProtocol.ID("test/protocol1")

	peerId := peer.ID("peerId")

	//
	slots.GetPeers(protocol).Add(peerId)
	slots.GetPeers(protocol1).Add(peerId)
	//
	fetchedPeer, err := slots.GetPeers(protocol1).GetRandom()
	require.NoError(t, err)
	require.Equal(t, peerId, fetchedPeer)

	//
	slots.RemovePeer(peerId)
	//
	_, err = slots.GetPeers(protocol).GetRandom()
	require.Equal(t, err, utils.ErrNoPeersAvailable)
	_, err = slots.GetPeers(protocol1).GetRandom()
	require.Equal(t, err, utils.ErrNoPeersAvailable)
}
