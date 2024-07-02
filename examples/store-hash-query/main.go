package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/store"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var log = utils.Logger().Named("store-hash-query")

func randomHex(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func main() {
	ctx := context.Background()

	key, err := randomHex(32)
	if err != nil {
		log.Error("Could not generate random key", zap.Error(err))
		return
	}
	prvKey, err := crypto.HexToECDSA(key)
	if err != nil {
		log.Error("Could not convert hex into ecdsa key", zap.Error(err))
		return
	}
	hostAddr, _ := net.ResolveTCPAddr("tcp", "0.0.0.0:0")
	wakuNode, err := node.New(
		node.WithPrivateKey(prvKey),
		node.WithHostAddress(hostAddr),
		node.WithNTP(),
		node.WithWakuStore(),
		node.WithClusterID(uint16(16)),
		node.WithLogLevel(zapcore.DebugLevel),
	)
	if err != nil {
		log.Error("Error creating wakunode", zap.Error(err))
		return
	}
	err = wakuNode.Start(ctx)
	if err != nil {
		log.Error("Error starting wakunode", zap.Error(err))
		return
	}

	time.Sleep(3 * time.Second)

	addr, err := peer.AddrInfoFromString("/dns4/store-02.ac-cn-hongkong-c.shards.test.status.im/tcp/30303/p2p/16Uiu2HAm9CQhsuwPR54q27kNj9iaQVfyRzTGKrhFmr94oD8ujU6P")
	if err != nil {
		log.Error("Error parsing peer address", zap.Error(err))
		return
	}
	wakuNode.DialPeerWithInfo(ctx, *addr)

	time.Sleep(3 * time.Second)

	msgHash, err := hexutil.Decode("0x081bf9e247a2c892a67a1e063ec2e7b4bb367e67e53aefce43e0a6465b0d3671")
	if err != nil {
		log.Error("Error decoding message hash", zap.Error(err))
		return
	}
	var b pb.MessageHash
	copy(b[:], msgHash)

	var opts []store.RequestOption
	requestID := protocol.GenerateRequestID()
	opts = append(opts, store.WithRequestID(requestID))
	opts = append(opts, store.WithPeer(addr.ID))

	result, err := wakuNode.Store().QueryByHash(ctx, []pb.MessageHash{b}, opts...)
	if err != nil {
		log.Warn("store.queryByHash failed", zap.String("requestID", hexutil.Encode(requestID)), zap.Error(err))
		return
	}

	log.Info("store.queryByHash result", zap.String("requestID", hexutil.Encode(requestID)), zap.Any("messages", result.Messages()))

	if len(result.Messages()) == 0 {
		log.Warn("store.queryByHash result empty")
		return
	}
	resHash := result.Messages()[0].MessageHash
	log.Info("store.queryByHash result hash", zap.String("hash", hexutil.Encode(resHash[:])))
}
