package relay

import (
	"context"
	"crypto/rand"
	"testing"

	pubsub_pb "github.com/libp2p/go-libp2p-pubsub/pb"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/timesource"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

func TestWakuRelay(t *testing.T) {
	testTopic := "/waku/2/go/relay/test"

	port, err := tests.FindFreePort(t, "", 5)
	require.NoError(t, err)

	host, err := tests.MakeHost(context.Background(), port, rand.Reader)
	require.NoError(t, err)

	relay := NewWakuRelay(host, nil, 0, timesource.NewDefaultClock(), utils.Logger())
	err = relay.Start(context.Background())
	require.NoError(t, err)
	defer relay.Stop()

	sub, err := relay.subscribe(testTopic)
	defer sub.Cancel()
	require.NoError(t, err)

	topics := relay.Topics()
	require.Equal(t, 1, len(topics))
	require.Equal(t, testTopic, topics[0])

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		defer cancel()

		_, err := sub.Next(ctx)
		require.NoError(t, err)
	}()

	msg := &pb.WakuMessage{
		Payload:      []byte{1},
		Version:      0,
		ContentTopic: "test",
		Timestamp:    0,
	}
	_, err = relay.PublishToTopic(context.Background(), msg, testTopic)
	require.NoError(t, err)

	<-ctx.Done()
}

func TestMsgID(t *testing.T) {
	expectedMsgIdBytes := []byte{208, 214, 63, 55, 144, 6, 206, 39, 40, 251, 138, 74, 66, 168, 43, 32, 91, 94, 149, 122, 237, 198, 149, 87, 232, 156, 197, 34, 53, 131, 78, 112}

	topic := "abcde"
	msg := &pubsub_pb.Message{
		Data:  []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11},
		Topic: &topic,
	}

	msgId := msgIdFn(msg)

	require.Equal(t, expectedMsgIdBytes, []byte(msgId))
}
