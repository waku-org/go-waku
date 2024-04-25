package storev3

//111go:build include_storev3_tests
// 111+build include_storev3_tests

import (
	"context"
	"crypto/rand"
	"os"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/peermanager"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/timesource"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"google.golang.org/protobuf/proto"
)

func TestStoreClient(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	port, err := tests.FindFreePort(t, "", 5)
	require.NoError(t, err)

	host, err := tests.MakeHost(context.Background(), port, rand.Reader)
	require.NoError(t, err)

	// Creating a storeV3 instance for all queries
	pm := peermanager.NewPeerManager(5, 5, utils.Logger())
	pm.SetHost(host)
	wakuStore := NewWakuStoreV3(pm, timesource.NewDefaultClock(), utils.Logger())
	wakuStore.SetHost(host)

	// Creating a relay instance for pushing messages to the store node
	b := relay.NewBroadcaster(10)
	require.NoError(t, b.Start(context.Background()))
	wakuRelay := relay.NewWakuRelay(b, 0, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, utils.Logger())
	wakuRelay.SetHost(host)
	require.NoError(t, err)
	err = wakuRelay.Start(context.Background())
	require.NoError(t, err)

	_, err = wakuRelay.Subscribe(context.Background(), protocol.NewContentFilter(protocol.DefaultPubsubTopic{}.String()), relay.WithoutConsumer())
	require.NoError(t, err)

	// Obtain multiaddr from env
	envStorenode := os.Getenv("TEST_STOREV3_NODE")
	if envStorenode == "" {
		envStorenode = "/ip4/127.0.0.1/tcp/60000/p2p/16Uiu2HAmMGhfSTUzKbsjMWxc6T1X4wiTWSF1bEWSLjAukCm7KiHV"
	}
	storenode_multiaddr, err := multiaddr.NewMultiaddr(envStorenode)
	require.NoError(t, err)

	storenode, err := peer.AddrInfoFromP2pAddr(storenode_multiaddr)
	require.NoError(t, err)

	err = host.Connect(ctx, *storenode)
	require.NoError(t, err)

	// Wait until mesh forms
	time.Sleep(2 * time.Second)

	// Send messages
	messages := []*pb.WakuMessage{}
	startTime := utils.GetUnixEpoch(timesource.NewDefaultClock())
	for i := 0; i < 5; i++ {
		msg := &pb.WakuMessage{
			Payload:      []byte{1, 2, 3, 4, 5},
			ContentTopic: "test",
			Version:      proto.Uint32(0),
			Timestamp:    utils.GetUnixEpoch(timesource.NewDefaultClock()),
		}
		_, err := wakuRelay.Publish(ctx, msg, relay.WithDefaultPubsubTopic())
		require.NoError(t, err)

		messages = append(messages, msg)
		time.Sleep(20 * time.Millisecond)
	}
	endTime := utils.GetUnixEpoch(timesource.NewDefaultClock())

	time.Sleep(1 * time.Second)

	// Check for message existence
	exists, err := wakuStore.Exists(ctx, messages[0].Hash(relay.DefaultWakuTopic), WithPeer(storenode.ID))
	require.NoError(t, err)
	require.True(t, exists)

	// Message should not exist
	exists, err = wakuStore.Exists(ctx, pb.MessageHash{1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2}, WithPeer(storenode.ID))
	require.NoError(t, err)
	require.False(t, exists)

	// Query messages with forward pagination
	response, err := wakuStore.Query(ctx, FilterCriteria{ContentFilter: protocol.NewContentFilter(relay.DefaultWakuTopic, "test"), TimeStart: startTime, TimeEnd: endTime}, WithPaging(true, 2))
	require.NoError(t, err)

	// -- First page:
	hasNext, err := response.Next(ctx)
	require.NoError(t, err)
	require.True(t, hasNext)
	require.False(t, response.IsComplete())
	require.Len(t, response.messages, 2)
	require.Equal(t, response.messages[0].Message.GetTimestamp(), messages[0].GetTimestamp())
	require.Equal(t, response.messages[1].Message.GetTimestamp(), messages[1].GetTimestamp())

	// -- Second page:
	hasNext, err = response.Next(ctx)
	require.NoError(t, err)
	require.True(t, hasNext)
	require.False(t, response.IsComplete())
	require.Len(t, response.messages, 2)
	require.Equal(t, response.messages[0].Message.GetTimestamp(), messages[2].GetTimestamp())
	require.Equal(t, response.messages[1].Message.GetTimestamp(), messages[3].GetTimestamp())

	// -- Third page:
	hasNext, err = response.Next(ctx)
	require.NoError(t, err)
	require.False(t, hasNext)
	require.True(t, response.IsComplete())
	require.Len(t, response.messages, 1)
	require.Equal(t, response.messages[0].Message.GetTimestamp(), messages[4].GetTimestamp())

	// -- Trying to continue a completed cursor
	hasNext, err = response.Next(ctx)
	require.NoError(t, err)
	require.False(t, hasNext)
	require.True(t, response.IsComplete())
	require.Len(t, response.messages, 0)

	// Query messages with backward pagination
	response, err = wakuStore.Query(ctx, FilterCriteria{ContentFilter: protocol.NewContentFilter(relay.DefaultWakuTopic, "test"), TimeStart: startTime, TimeEnd: endTime}, WithPaging(false, 2))
	require.NoError(t, err)

	// -- First page:
	hasNext, err = response.Next(ctx)
	require.NoError(t, err)
	require.True(t, hasNext)
	require.False(t, response.IsComplete())
	require.Len(t, response.messages, 2)
	require.Equal(t, response.messages[0].Message.GetTimestamp(), messages[3].GetTimestamp())
	require.Equal(t, response.messages[1].Message.GetTimestamp(), messages[4].GetTimestamp())

	// -- Second page:
	hasNext, err = response.Next(ctx)
	require.NoError(t, err)
	require.True(t, hasNext)
	require.False(t, response.IsComplete())
	require.Len(t, response.messages, 2)
	require.Equal(t, response.messages[0].Message.GetTimestamp(), messages[1].GetTimestamp())
	require.Equal(t, response.messages[1].Message.GetTimestamp(), messages[2].GetTimestamp())

	// -- Third page:
	hasNext, err = response.Next(ctx)
	require.NoError(t, err)
	require.False(t, hasNext)
	require.True(t, response.IsComplete())
	require.Len(t, response.messages, 1)
	require.Equal(t, response.messages[0].Message.GetTimestamp(), messages[0].GetTimestamp())

	// -- Trying to continue a completed cursor
	hasNext, err = response.Next(ctx)
	require.NoError(t, err)
	require.False(t, hasNext)
	require.True(t, response.IsComplete())
	require.Len(t, response.messages, 0)

	// No cursor should be returned if there are no messages that match the criteria
	response, err = wakuStore.Query(ctx, FilterCriteria{ContentFilter: protocol.NewContentFilter(relay.DefaultWakuTopic, "no-messages"), TimeStart: startTime, TimeEnd: endTime}, WithPaging(true, 2))
	require.NoError(t, err)
	require.Len(t, response.messages, 0)
	require.Empty(t, response.Cursor())

	// If the page size is larger than the number of existing messages, it should not return a cursor
	response, err = wakuStore.Query(ctx, FilterCriteria{ContentFilter: protocol.NewContentFilter(relay.DefaultWakuTopic, "test"), TimeStart: startTime, TimeEnd: endTime}, WithPaging(true, 100))
	require.NoError(t, err)
	require.Len(t, response.messages, 5)
	require.Empty(t, response.Cursor())

	// Invalid cursors should return an empty response
	// TODO: nwaku is returning values even with invalid cursors
	/*
		response, err = wakuStore.Query(ctx, FilterCriteria{ContentFilter: protocol.NewContentFilter(relay.DefaultWakuTopic, "test"), TimeStart: startTime, TimeEnd: endTime}, WithCursor([]byte{1, 2, 3, 4, 5, 6}))
		require.NoError(t, err)
		require.Len(t, response.messages, 0)
		require.Empty(t, response.Cursor())
	*/

	// Handle temporal history query with an invalid time window
	_, err = wakuStore.Query(ctx, FilterCriteria{ContentFilter: protocol.NewContentFilter(relay.DefaultWakuTopic, "test"), TimeStart: endTime, TimeEnd: startTime})
	require.NotNil(t, err)

	// Handle temporal history query with a zero-size time window
	response, err = wakuStore.Query(ctx, FilterCriteria{ContentFilter: protocol.NewContentFilter(relay.DefaultWakuTopic, "test"), TimeStart: startTime, TimeEnd: startTime})
	require.NoError(t, err)
	require.Len(t, response.messages, 0)
	require.Empty(t, response.Cursor())

	// Should not include data
	// TODO: nwaku is returning the data always
	/*
		response, err = wakuStore.Request(ctx, MessageHashCriteria{MessageHashes: []pb.MessageHash{messages[0].Hash(relay.DefaultWakuTopic)}}, IncludeData(false), WithPeer(storenode.ID))
		require.NoError(t, err)
		require.Len(t, response.messages, 1)
		require.Nil(t, response.messages[0].Message)

		response, err = wakuStore.Request(ctx, FilterCriteria{ContentFilter: protocol.NewContentFilter(relay.DefaultWakuTopic, "test")}, IncludeData(false))
		require.NoError(t, err)
		require.GreaterOrEqual(t, response.messages, 1)
		require.Nil(t, response.messages[0].Message)
	*/
}
