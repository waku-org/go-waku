package tests

import (
	"context"
	"net"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	"github.com/status-im/go-waku/waku/v2/node"
	"github.com/status-im/go-waku/waku/v2/protocol/pb"
	"github.com/status-im/go-waku/waku/v2/utils"
)

func TestBasicSendingReceiving(t *testing.T) {
	hostAddr, err := net.ResolveTCPAddr("tcp", "0.0.0.0:0")
	require.NoError(t, err)

	key, err := RandomHex(32)
	require.NoError(t, err)

	prvKey, err := crypto.HexToECDSA(key)
	require.NoError(t, err)

	ctx := context.Background()

	wakuNode, err := node.New(ctx,
		node.WithPrivateKey(prvKey),
		node.WithHostAddress([]*net.TCPAddr{hostAddr}),
		node.WithWakuRelay(),
	)
	require.NoError(t, err)
	require.NotNil(t, wakuNode)

	err = wakuNode.Start()
	require.NoError(t, err)

	require.NoError(t, write(ctx, wakuNode, "test"))

	sub, err := wakuNode.Relay().Subscribe(ctx, nil)
	require.NoError(t, err)

	value := <-sub.C
	payload, err := node.DecodePayload(value.Message(), &node.KeyInfo{Kind: node.None})
	require.NoError(t, err)

	require.Contains(t, string(payload.Data), "test")
}

func write(ctx context.Context, wakuNode *node.WakuNode, msgContent string) error {
	var contentTopic string = "test"
	var version uint32 = 0
	var timestamp float64 = utils.GetUnixEpoch()

	p := new(node.Payload)
	p.Data = []byte(wakuNode.ID() + ": " + msgContent)
	p.Key = &node.KeyInfo{Kind: node.None}

	payload, err := p.Encode(version)
	if err != nil {
		return err
	}

	msg := &pb.WakuMessage{
		Payload:      payload,
		Version:      version,
		ContentTopic: contentTopic,
		Timestamp:    timestamp,
	}

	_, err = wakuNode.Relay().Publish(ctx, msg, nil)
	return err
}
