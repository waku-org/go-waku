package relay

import (
	"bytes"
	"context"
	"encoding/hex"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pubsub_pb "github.com/libp2p/go-libp2p-pubsub/pb"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/timesource"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"google.golang.org/protobuf/proto"
)

func TestMsgHash(t *testing.T) {
	privKeyB, _ := hex.DecodeString("5526a8990317c9b7b58d07843d270f9cd1d9aaee129294c1c478abf7261dd9e6")
	prvKey, _ := crypto.ToECDSA(privKeyB)

	payload, _ := hex.DecodeString("1A12E077D0E89F9CAC11FBBB6A676C86120B5AD3E248B1F180E98F15EE43D2DFCF62F00C92737B2FF6F59B3ABA02773314B991C41DC19ADB0AD8C17C8E26757B")
	contentTopic := "content-topic"
	pubsubTopic := "pubsub-topic"

	msg := &pb.WakuMessage{
		Payload:      payload,
		ContentTopic: contentTopic,
		Timestamp:    utils.GetUnixEpoch(timesource.NewDefaultClock()),
	}

	err := SignMessage(prvKey, pubsubTopic, msg)
	require.NoError(t, err)

	expectedSignature, _ := hex.DecodeString("B139487797A242291E0DD3F689777E559FB749D565D55FF202C18E24F21312A555043437B4F808BB0D21D542D703873DC712D76A3BAF1C5C8FF754210D894AD4")
	require.True(t, bytes.Equal(expectedSignature, msg.Meta))

	msgData, _ := proto.Marshal(msg)

	expectedMessageHash, _ := hex.DecodeString("0914369D6D0C13783A8E86409FE42C68D8E8296456B9A9468C845006BCE5B9B2")
	messageHash := MsgHash(pubsubTopic, msg)
	require.True(t, bytes.Equal(expectedMessageHash, messageHash))

	myValidator := validatorFnBuilder(pubsubTopic, &prvKey.PublicKey)

	result := myValidator(context.Background(), "", &pubsub.Message{
		Message: &pubsub_pb.Message{
			Data: msgData,
		},
	})
	require.True(t, result)
}
