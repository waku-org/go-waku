package relay

import (
	"bytes"
	"context"
	"encoding/hex"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pubsub_pb "github.com/libp2p/go-libp2p-pubsub/pb"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"google.golang.org/protobuf/proto"
)

type FakeTimesource struct {
	now time.Time
}

func NewFakeTimesource(now time.Time) FakeTimesource {
	return FakeTimesource{now}
}

func (f FakeTimesource) Start(ctx context.Context) error {
	return nil
}

func (f FakeTimesource) Stop() {}

func (f FakeTimesource) Now() time.Time {
	return f.now
}

type Timesource interface {
	Now() time.Time
}

func TestProtectedTopicValidator(t *testing.T) {
	privKeyB, _ := hex.DecodeString("5526a8990317c9b7b58d07843d270f9cd1d9aaee129294c1c478abf7261dd9e6")
	prvKey, _ := crypto.ToECDSA(privKeyB)

	payload, _ := hex.DecodeString("1A12E077D0E89F9CAC11FBBB6A676C86120B5AD3E248B1F180E98F15EE43D2DFCF62F00C92737B2FF6F59B3ABA02773314B991C41DC19ADB0AD8C17C8E26757B")
	contentTopic := "content-topic"
	ephemeral := true
	timestamp := time.Unix(0, 1683208172339052800)
	address := crypto.PubkeyToAddress(prvKey.PublicKey)

	protectedPubsubTopic := PrivKeyToTopic(prvKey, "proto")
	expectedPubsubTopic := "/waku/2/signed:0x2ea1f2ec2da14e0e3118e6c5bdfa0631785c76cd/proto"
	require.Equal(t, protectedPubsubTopic, expectedPubsubTopic)

	msg := &pb.WakuMessage{
		Payload:      payload,
		ContentTopic: contentTopic,
		Timestamp:    timestamp.UnixNano(),
		Ephemeral:    ephemeral,
	}

	err := SignMessage(prvKey, msg, protectedPubsubTopic)
	require.NoError(t, err)

	expectedSignature, _ := hex.DecodeString("7FB40AB47AA92A6395137C4B477C2D028C6F0F0E7A712D993EDAC5021E5B285E16956E2FD2421261D718D5C945A92600E0F8140AB7ADCF988178BF1AE577FAD901")
	require.True(t, bytes.Equal(expectedSignature, msg.Meta))

	msgData, _ := proto.Marshal(msg)

	expectedMessageHash, _ := hex.DecodeString("C3C74531E446CB5D0681A1919E643CCD304A7FA23A299691945BADA75A55C04C")
	messageHash := MsgHash(protectedPubsubTopic, msg)
	require.True(t, bytes.Equal(expectedMessageHash, messageHash))

	myValidator, err := validatorFnBuilder(NewFakeTimesource(timestamp), protectedPubsubTopic, address)
	require.NoError(t, err)
	result := myValidator(context.Background(), "", &pubsub.Message{
		Message: &pubsub_pb.Message{
			Data: msgData,
		},
	})
	require.True(t, result)

	// Exceed 5m window in both directions
	now5m1sInPast := timestamp.Add(-5 * time.Minute).Add(-1 * time.Second)
	myValidator, err = validatorFnBuilder(NewFakeTimesource(now5m1sInPast), protectedPubsubTopic, address)
	require.NoError(t, err)
	result = myValidator(context.Background(), "", &pubsub.Message{
		Message: &pubsub_pb.Message{
			Data: msgData,
		},
	})
	require.False(t, result)

	now5m1sInFuture := timestamp.Add(5 * time.Minute).Add(1 * time.Second)
	myValidator, err = validatorFnBuilder(NewFakeTimesource(now5m1sInFuture), protectedPubsubTopic, address)
	require.NoError(t, err)
	result = myValidator(context.Background(), "", &pubsub.Message{
		Message: &pubsub_pb.Message{
			Data: msgData,
		},
	})
	require.False(t, result)
}
