package tests

import (
	"context"
	"net"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-waku/waku/v2/payload"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

func TestBasicSendingReceiving(t *testing.T) {
	hostAddr, err := net.ResolveTCPAddr("tcp", "0.0.0.0:0")
	require.NoError(t, err)

	key, err := RandomHex(32)
	require.NoError(t, err)

	prvKey, err := crypto.HexToECDSA(key)
	require.NoError(t, err)

	ctx := context.Background()

	wakuNode, err := node.New(
		node.WithPrivateKey(prvKey),
		node.WithHostAddress(hostAddr),
		node.WithWakuRelay(),
	)
	require.NoError(t, err)
	require.NotNil(t, wakuNode)

	err = wakuNode.Start(ctx)
	require.NoError(t, err)

	require.NoError(t, write(ctx, wakuNode, "test"))

	sub, err := wakuNode.Relay().Subscribe(ctx, protocol.NewContentFilter(relay.DefaultWakuTopic))
	require.NoError(t, err)

	value := <-sub[0].Ch
	payload, err := payload.DecodePayload(value.Message(), &payload.KeyInfo{Kind: payload.None})
	require.NoError(t, err)

	require.Contains(t, string(payload.Data), "test")
}

func write(ctx context.Context, wakuNode *node.WakuNode, msgContent string) error {
	var contentTopic string = "test"
	version := uint32(0)
	timestamp := utils.GetUnixEpoch()

	p := new(payload.Payload)
	p.Data = []byte(wakuNode.ID() + ": " + msgContent)
	p.Key = &payload.KeyInfo{Kind: payload.None}

	payload, err := p.Encode(version)
	if err != nil {
		return err
	}

	msg := &pb.WakuMessage{
		Payload:      payload,
		Version:      proto.Uint32(version),
		ContentTopic: contentTopic,
		Timestamp:    timestamp,
	}

	_, err = wakuNode.Relay().Publish(ctx, msg, relay.WithDefaultPubsubTopic())
	return err
}
