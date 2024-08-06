package publish

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"google.golang.org/protobuf/proto"
)

func TestFifoQueue(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	queue := NewMessageQueue(10, false)
	go queue.Start(ctx)

	queue.Push(ctx, protocol.NewEnvelope(&pb.WakuMessage{}, 0, "A"))
	queue.Push(ctx, protocol.NewEnvelope(&pb.WakuMessage{}, 0, "B"))
	queue.Push(ctx, protocol.NewEnvelope(&pb.WakuMessage{}, 0, "C"))

	envelope, ok := <-queue.Pop(ctx)
	require.True(t, ok)
	require.Equal(t, "A", envelope.PubsubTopic())

	envelope, ok = <-queue.Pop(ctx)
	require.True(t, ok)
	require.Equal(t, "B", envelope.PubsubTopic())

	envelope, ok = <-queue.Pop(ctx)
	require.True(t, ok)
	require.Equal(t, "C", envelope.PubsubTopic())

	cancel()

	_, ok = <-queue.Pop(ctx)
	require.False(t, ok)
}

func TestPriorityQueue(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	queue := NewMessageQueue(10, true)
	go queue.Start(ctx)

	queue.Push(ctx, protocol.NewEnvelope(&pb.WakuMessage{Timestamp: proto.Int64(0)}, 0, "A"), LowPriority)
	queue.Push(ctx, protocol.NewEnvelope(&pb.WakuMessage{Timestamp: proto.Int64(1)}, 0, "B"), LowPriority)
	queue.Push(ctx, protocol.NewEnvelope(&pb.WakuMessage{Timestamp: proto.Int64(2)}, 0, "C"), HighPriority)
	queue.Push(ctx, protocol.NewEnvelope(&pb.WakuMessage{Timestamp: proto.Int64(3)}, 0, "D"), NormalPriority)
	queue.Push(ctx, protocol.NewEnvelope(&pb.WakuMessage{Timestamp: proto.Int64(4)}, 0, "E"), HighPriority)
	queue.Push(ctx, protocol.NewEnvelope(&pb.WakuMessage{Timestamp: proto.Int64(5)}, 0, "F"), LowPriority)
	queue.Push(ctx, protocol.NewEnvelope(&pb.WakuMessage{Timestamp: proto.Int64(6)}, 0, "G"), NormalPriority)

	time.Sleep(2 * time.Second)

	envelope, ok := <-queue.Pop(ctx)
	require.True(t, ok)
	require.Equal(t, "C", envelope.PubsubTopic())

	envelope, ok = <-queue.Pop(ctx)
	require.True(t, ok)
	require.Equal(t, "E", envelope.PubsubTopic())

	envelope, ok = <-queue.Pop(ctx)
	require.True(t, ok)
	require.Equal(t, "D", envelope.PubsubTopic())

	envelope, ok = <-queue.Pop(ctx)
	require.True(t, ok)
	require.Equal(t, "G", envelope.PubsubTopic())

	envelope, ok = <-queue.Pop(ctx)
	require.True(t, ok)
	require.Equal(t, "A", envelope.PubsubTopic())

	envelope, ok = <-queue.Pop(ctx)
	require.True(t, ok)
	require.Equal(t, "B", envelope.PubsubTopic())

	envelope, ok = <-queue.Pop(ctx)
	require.True(t, ok)
	require.Equal(t, "F", envelope.PubsubTopic())

	cancel()

	_, ok = <-queue.Pop(ctx)
	require.False(t, ok)

}
