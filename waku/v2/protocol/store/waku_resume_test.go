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

func TestFindLastSeenMessage(t *testing.T) {
	msg1 := &pb.WakuMessage{Payload: []byte{1, 2, 3}, ContentTopic: "1", Version: 0, Timestamp: float64(1)}
	msg2 := &pb.WakuMessage{Payload: []byte{1, 2, 3}, ContentTopic: "2", Version: 0, Timestamp: float64(2)}
	msg3 := &pb.WakuMessage{Payload: []byte{1, 2, 3}, ContentTopic: "3", Version: 0, Timestamp: float64(3)}
	msg4 := &pb.WakuMessage{Payload: []byte{1, 2, 3}, ContentTopic: "4", Version: 0, Timestamp: float64(4)}
	msg5 := &pb.WakuMessage{Payload: []byte{1, 2, 3}, ContentTopic: "5", Version: 0, Timestamp: float64(5)}

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

	msg0 := &pb.WakuMessage{Payload: []byte{1, 2, 3}, ContentTopic: "2", Version: 0, Timestamp: float64(0 * time.Second)}
	msg1 := &pb.WakuMessage{Payload: []byte{1, 2, 3}, ContentTopic: "1", Version: 0, Timestamp: float64(1 * time.Second)}
	msg2 := &pb.WakuMessage{Payload: []byte{1, 2, 3}, ContentTopic: "2", Version: 0, Timestamp: float64(2 * time.Second)}
	msg3 := &pb.WakuMessage{Payload: []byte{1, 2, 3}, ContentTopic: "1", Version: 0, Timestamp: float64(3 * time.Second)}
	msg4 := &pb.WakuMessage{Payload: []byte{1, 2, 3}, ContentTopic: "2", Version: 0, Timestamp: float64(4 * time.Second)}
	msg5 := &pb.WakuMessage{Payload: []byte{1, 2, 3}, ContentTopic: "1", Version: 0, Timestamp: float64(5 * time.Second)}
	msg6 := &pb.WakuMessage{Payload: []byte{1, 2, 3}, ContentTopic: "2", Version: 0, Timestamp: float64(6 * time.Second)}
	msg7 := &pb.WakuMessage{Payload: []byte{1, 2, 3}, ContentTopic: "1", Version: 0, Timestamp: float64(7 * time.Second)}
	msg8 := &pb.WakuMessage{Payload: []byte{1, 2, 3}, ContentTopic: "2", Version: 0, Timestamp: float64(8 * time.Second)}
	msg9 := &pb.WakuMessage{Payload: []byte{1, 2, 3}, ContentTopic: "1", Version: 0, Timestamp: float64(9 * time.Second)}

	s1.storeMessage("test", msg0)
	s1.storeMessage("test", msg1)
	s1.storeMessage("test", msg2)
	s1.storeMessage("test", msg3)
	s1.storeMessage("test", msg4)
	s1.storeMessage("test", msg5)
	s1.storeMessage("test", msg6)
	s1.storeMessage("test", msg7)
	s1.storeMessage("test", msg8)
	s1.storeMessage("test", msg9)

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
