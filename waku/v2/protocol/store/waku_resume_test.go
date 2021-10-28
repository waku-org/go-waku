package store

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/status-im/go-waku/waku/v2/protocol/pb"
	"github.com/stretchr/testify/require"
)

func createWakuMessage(contentTopic string, timestamp float64) *pb.WakuMessage {
	return &pb.WakuMessage{Payload: []byte{1, 2, 3}, ContentTopic: contentTopic, Version: 0, Timestamp: timestamp}
}

func TestFindLastSeenMessage(t *testing.T) {
	msg1 := createWakuMessage("1", 1)
	msg2 := createWakuMessage("2", 2)
	msg3 := createWakuMessage("3", 3)
	msg4 := createWakuMessage("4", 4)
	msg5 := createWakuMessage("5", 5)

	s := NewWakuStore(true, nil)
	s.storeMessage("test", msg1)
	s.storeMessage("test", msg3)
	s.storeMessage("test", msg5)
	s.storeMessage("test", msg2)
	s.storeMessage("test", msg4)

	require.Equal(t, msg5.Timestamp, s.findLastSeen())
}

func TestResume(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	host1, err := libp2p.New(ctx, libp2p.DefaultTransports, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	require.NoError(t, err)

	s1 := NewWakuStore(true, nil)
	s1.Start(ctx, host1)
	defer s1.Stop()

	for i := 0; i < 10; i++ {
		var contentTopic = "1"
		if i%2 == 0 {
			contentTopic = "2"
		}

		msg := createWakuMessage(contentTopic, float64(time.Duration(i)*time.Second))
		s1.storeMessage("test", msg)
	}

	host2, err := libp2p.New(ctx, libp2p.DefaultTransports, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	require.NoError(t, err)

	s2 := NewWakuStore(false, nil)
	s2.Start(ctx, host2)
	defer s2.Stop()

	host2.Peerstore().AddAddr(host1.ID(), getHostAddress(host1), peerstore.PermanentAddrTTL)
	err = host2.Peerstore().AddProtocols(host1.ID(), string(StoreID_v20beta3))
	require.NoError(t, err)

	msgCount, err := s2.Resume(ctx, "test", []peer.ID{host1.ID()})

	require.NoError(t, err)
	require.Equal(t, 10, msgCount)
	require.Len(t, s2.messages, 10)

	// Test duplication
	msgCount, err = s2.Resume(ctx, "test", []peer.ID{host1.ID()})

	require.NoError(t, err)
	require.Equal(t, 0, msgCount)
}

func TestResumeWithListOfPeers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Host that does not support store protocol
	invalidHost, err := libp2p.New(ctx, libp2p.DefaultTransports, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	require.NoError(t, err)

	host1, err := libp2p.New(ctx, libp2p.DefaultTransports, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	require.NoError(t, err)

	s1 := NewWakuStore(true, nil)
	s1.Start(ctx, host1)
	defer s1.Stop()

	msg0 := &pb.WakuMessage{Payload: []byte{1, 2, 3}, ContentTopic: "2", Version: 0, Timestamp: float64(0 * time.Second)}

	s1.storeMessage("test", msg0)

	host2, err := libp2p.New(ctx, libp2p.DefaultTransports, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	require.NoError(t, err)

	s2 := NewWakuStore(false, nil)
	s2.Start(ctx, host2)
	defer s2.Stop()

	host2.Peerstore().AddAddr(host1.ID(), getHostAddress(host1), peerstore.PermanentAddrTTL)
	err = host2.Peerstore().AddProtocols(host1.ID(), string(StoreID_v20beta3))
	require.NoError(t, err)

	msgCount, err := s2.Resume(ctx, "test", []peer.ID{invalidHost.ID(), host1.ID()})

	require.NoError(t, err)
	require.Equal(t, 1, msgCount)
	require.Len(t, s2.messages, 1)
}

func TestResumeWithoutSpecifyingPeer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	host1, err := libp2p.New(ctx, libp2p.DefaultTransports, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	require.NoError(t, err)

	s1 := NewWakuStore(true, nil)
	s1.Start(ctx, host1)
	defer s1.Stop()

	msg0 := &pb.WakuMessage{Payload: []byte{1, 2, 3}, ContentTopic: "2", Version: 0, Timestamp: float64(0 * time.Second)}

	s1.storeMessage("test", msg0)

	host2, err := libp2p.New(ctx, libp2p.DefaultTransports, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	require.NoError(t, err)

	s2 := NewWakuStore(false, nil)
	s2.Start(ctx, host2)
	defer s2.Stop()

	host2.Peerstore().AddAddr(host1.ID(), getHostAddress(host1), peerstore.PermanentAddrTTL)
	err = host2.Peerstore().AddProtocols(host1.ID(), string(StoreID_v20beta3))
	require.NoError(t, err)

	msgCount, err := s2.Resume(ctx, "test", []peer.ID{})

	require.NoError(t, err)
	require.Equal(t, 1, msgCount)
	require.Len(t, s2.messages, 1)
}
