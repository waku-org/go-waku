package rpc

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/status-im/go-waku/waku/v2/node"
	"github.com/status-im/go-waku/waku/v2/utils"
	"github.com/stretchr/testify/require"
)

func makePrivateService(t *testing.T) *PrivateService {
	n, err := node.New(context.Background(), node.WithWakuRelayAndMinPeers(0))
	require.NoError(t, err)
	err = n.Start()
	require.NoError(t, err)

	return NewPrivateService(n, utils.Logger())
}

func TestGetV1SymmetricKey(t *testing.T) {
	d := makePrivateService(t)
	defer d.node.Stop()

	var reply SymmetricKeyReply
	err := d.GetV1SymmetricKey(
		makeRequest(t),
		&Empty{},
		&reply,
	)
	require.NoError(t, err)
	require.NotEmpty(t, reply)
}

func TestGetV1AsymmetricKey(t *testing.T) {
	d := makePrivateService(t)
	defer d.node.Stop()

	var reply KeyPairReply
	err := d.GetV1AsymmetricKeypair(
		makeRequest(t),
		&Empty{},
		&reply,
	)
	require.NoError(t, err)
	require.NotEmpty(t, reply.PublicKey)
	require.NotEmpty(t, reply.PrivateKey)
}

func TestPostV1SymmetricMessage(t *testing.T) {
	d := makePrivateService(t)
	defer d.node.Stop()

	var reply SuccessReply
	err := d.PostV1SymmetricMessage(
		makeRequest(t),
		&SymmetricMessageArgs{
			Topic:   "test",
			Message: RPCWakuMessage{Payload: []byte("test")},
			SymKey:  "0x1122334455667788991011223344556677889910112233445566778899101122",
		},
		&reply,
	)
	require.NoError(t, err)
	require.True(t, reply)
}

func TestPostV1AsymmetricMessage(t *testing.T) {
	d := makePrivateService(t)
	defer d.node.Stop()

	var reply bool
	err := d.PostV1AsymmetricMessage(
		makeRequest(t),
		&AsymmetricMessageArgs{
			Topic:     "test",
			Message:   RPCWakuMessage{Payload: []byte("test")},
			PublicKey: "0x045ded6a56c88173e87a88c55b96956964b1bd3351b5fcb70950a4902fbc1bc0ceabb0ac846c3a4b8f2f6024c0e19f0a7f6a4865035187de5463f34012304fc7c5",
		},
		&reply,
	)
	require.NoError(t, err)
	require.True(t, reply)
}

func TestGetV1SymmetricMessages(t *testing.T) {
	d := makePrivateService(t)
	go d.Start()
	defer d.node.Stop()

	// Subscribing topic to test getter
	_, err := d.node.Relay().SubscribeToTopic(context.TODO(), "test")
	require.NoError(t, err)

	var reply SuccessReply
	err = d.PostV1SymmetricMessage(
		makeRequest(t),
		&SymmetricMessageArgs{
			Topic:   "test",
			Message: RPCWakuMessage{Payload: []byte("test")},
			SymKey:  "0x1122334455667788991011223344556677889910112233445566778899101122",
		},
		&reply,
	)
	require.NoError(t, err)
	require.True(t, reply)

	time.Sleep(500 * time.Millisecond)

	var getReply MessagesReply
	err = d.GetV1SymmetricMessages(
		makeRequest(t),
		&SymmetricMessagesArgs{
			Topic:  "test",
			SymKey: "0x1122334455667788991011223344556677889910112233445566778899101122",
		},
		&getReply,
	)
	require.NoError(t, err)
	require.Len(t, getReply, 1)
}

func TestGetV1AsymmetricMessages(t *testing.T) {
	d := makePrivateService(t)
	go d.Start()
	defer d.node.Stop()

	// Subscribing topic to test getter
	_, err := d.node.Relay().SubscribeToTopic(context.TODO(), "test")
	require.NoError(t, err)

	prvKey, err := crypto.GenerateKey()
	require.NoError(t, err)

	var reply bool
	err = d.PostV1AsymmetricMessage(
		makeRequest(t),
		&AsymmetricMessageArgs{
			Topic:     "test",
			Message:   RPCWakuMessage{Payload: []byte("test")},
			PublicKey: hexutil.Encode(crypto.FromECDSAPub(&prvKey.PublicKey)),
		},
		&reply,
	)
	require.NoError(t, err)
	require.True(t, reply)

	time.Sleep(500 * time.Millisecond)

	var getReply MessagesReply
	err = d.GetV1AsymmetricMessages(
		makeRequest(t),
		&AsymmetricMessagesArgs{
			Topic:      "test",
			PrivateKey: hexutil.Encode(crypto.FromECDSA(prvKey)),
		},
		&getReply,
	)

	require.NoError(t, err)
	require.Len(t, getReply, 1)
}
