package relay

import (
	"bytes"
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pubsub_pb "github.com/libp2p/go-libp2p-pubsub/pb"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"google.golang.org/protobuf/proto"
)

func TestMsgHash(t *testing.T) {
	privKeyB, _ := hexutil.Decode("0x5526a8990317c9b7b58d07843d270f9cd1d9aaee129294c1c478abf7261dd9e6")
	prvKey, _ := crypto.ToECDSA(privKeyB)

	payload, _ := hexutil.Decode("0x3af5c7a8d71498e82e1991089d8429448f3b78277fac141af9052e77fc003dfb")
	contentTopic := "my-content-topic"
	pubsubTopic := "some-spam-protected-topic"

	msg := &pb.WakuMessage{
		Payload:      payload,
		ContentTopic: contentTopic,
	}

	err := SignMessage(prvKey, pubsubTopic, msg)
	require.NoError(t, err)

	msgData, _ := proto.Marshal(msg)

	expectedMessageHash, _ := hexutil.Decode("0xd0e3231ec48f9c0cf9306b7100c30b4e85c78854b67b41e4ee388fb4610f543d")
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
