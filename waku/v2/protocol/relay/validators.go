package relay

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/hex"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/waku-org/go-waku/waku/v2/hash"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"go.uber.org/zap"
	proto "google.golang.org/protobuf/proto"
)

// Application level message hash
func MsgHash(pubSubTopic string, msg *pb.WakuMessage) []byte {
	// TODO: Other fields?
	return hash.SHA256([]byte(pubSubTopic), msg.Payload, []byte(msg.ContentTopic))
}

type validatorFn = func(ctx context.Context, peerID peer.ID, message *pubsub.Message) bool

func validatorFnBuilder(topic string, publicKey *ecdsa.PublicKey) validatorFn {
	pubkBytes := crypto.FromECDSAPub(publicKey)
	return func(ctx context.Context, peerID peer.ID, message *pubsub.Message) bool {
		msg := new(pb.WakuMessage)
		err := proto.Unmarshal(message.Data, msg)
		if err != nil {
			return false
		}

		msgHash := MsgHash(topic, msg)
		signature := msg.Meta

		return secp256k1.VerifySignature(pubkBytes, msgHash, signature)
	}
}

func (w *WakuRelay) AddSignedTopicValidator(topic string, publicKey *ecdsa.PublicKey) error {
	w.log.Info("adding validator to signed topic", zap.String("topic", topic), zap.String("publicKey", hex.EncodeToString(elliptic.Marshal(publicKey.Curve, publicKey.X, publicKey.Y))))
	err := w.pubsub.RegisterTopicValidator(topic, validatorFnBuilder(topic, publicKey))
	if err != nil {
		return err
	}

	if !w.IsSubscribed(topic) {
		w.log.Warn("relay is not subscribed to signed topic", zap.String("topic", topic))
	}

	return nil
}

func SignMessage(privKey *ecdsa.PrivateKey, topic string, msg *pb.WakuMessage) error {
	msgHash := MsgHash(topic, msg)
	sign, err := secp256k1.Sign(msgHash, crypto.FromECDSA(privKey))
	if err != nil {
		return err
	}

	msg.Meta = sign[0:64] // Drop the V in R||S||V
	return nil
}
