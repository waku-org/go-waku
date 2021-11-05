package store

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/status-im/go-waku/tests"
	"github.com/status-im/go-waku/waku/v2/protocol"
	"github.com/status-im/go-waku/waku/v2/protocol/pb"
	"github.com/stretchr/testify/require"
)

func TestFindLastSeenMessage(t *testing.T) {
	msg1 := protocol.NewEnvelope(tests.CreateWakuMessage("1", 1), "test")
	msg2 := protocol.NewEnvelope(tests.CreateWakuMessage("2", 2), "test")
	msg3 := protocol.NewEnvelope(tests.CreateWakuMessage("3", 3), "test")
	msg4 := protocol.NewEnvelope(tests.CreateWakuMessage("4", 4), "test")
	msg5 := protocol.NewEnvelope(tests.CreateWakuMessage("5", 5), "test")

	s := NewWakuStore(nil)
	s.storeMessage(msg1)
	s.storeMessage(msg3)
	s.storeMessage(msg5)
	s.storeMessage(msg2)
	s.storeMessage(msg4)

	require.Equal(t, msg5.Message().Timestamp, s.findLastSeen())
}

func TestResume(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	host1, err := libp2p.New(ctx, libp2p.DefaultTransports, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	require.NoError(t, err)

	s1 := NewWakuStore(nil)
	s1.Start(ctx, host1)
	defer s1.Stop()

	for i := 0; i < 10; i++ {
		var contentTopic = "1"
		if i%2 == 0 {
			contentTopic = "2"
		}

		msg := protocol.NewEnvelope(tests.CreateWakuMessage(contentTopic, float64(time.Duration(i)*time.Second)), "test")
		s1.storeMessage(msg)
	}

	host2, err := libp2p.New(ctx, libp2p.DefaultTransports, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	require.NoError(t, err)

	s2 := NewWakuStore(nil)
	s2.Start(ctx, host2)
	defer s2.Stop()

	host2.Peerstore().AddAddr(host1.ID(), tests.GetHostAddress(host1), peerstore.PermanentAddrTTL)
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

	s1 := NewWakuStore(nil)
	s1.Start(ctx, host1)
	defer s1.Stop()

	msg0 := &pb.WakuMessage{Payload: []byte{1, 2, 3}, ContentTopic: "2", Version: 0, Timestamp: float64(0 * time.Second)}

	s1.storeMessage(protocol.NewEnvelope(msg0, "test"))

	host2, err := libp2p.New(ctx, libp2p.DefaultTransports, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	require.NoError(t, err)

	s2 := NewWakuStore(nil)
	s2.Start(ctx, host2)
	defer s2.Stop()

	host2.Peerstore().AddAddr(host1.ID(), tests.GetHostAddress(host1), peerstore.PermanentAddrTTL)
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

	s1 := NewWakuStore(nil)
	s1.Start(ctx, host1)
	defer s1.Stop()

	msg0 := &pb.WakuMessage{Payload: []byte{1, 2, 3}, ContentTopic: "2", Version: 0, Timestamp: float64(0 * time.Second)}

	s1.storeMessage(protocol.NewEnvelope(msg0, "test"))

	host2, err := libp2p.New(ctx, libp2p.DefaultTransports, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	require.NoError(t, err)

	s2 := NewWakuStore(nil)
	s2.Start(ctx, host2)
	defer s2.Stop()

	host2.Peerstore().AddAddr(host1.ID(), tests.GetHostAddress(host1), peerstore.PermanentAddrTTL)
	err = host2.Peerstore().AddProtocols(host1.ID(), string(StoreID_v20beta3))
	require.NoError(t, err)

	msgCount, err := s2.Resume(ctx, "test", []peer.ID{})

	require.NoError(t, err)
	require.Equal(t, 1, msgCount)
	require.Len(t, s2.messages, 1)
}
