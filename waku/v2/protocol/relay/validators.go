package relay

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/hex"

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

func (w *WakuRelay) AddSignedTopicValidator(topic string, publicKey *ecdsa.PublicKey) {
	w.log.Info("adding validator to signed topic", zap.String("topic", topic), zap.String("publicKey", hex.EncodeToString(elliptic.Marshal(publicKey.Curve, publicKey.X, publicKey.Y))))
	w.pubsub.RegisterTopicValidator(topic, func(ctx context.Context, peerID peer.ID, message *pubsub.Message) bool {
		msg := new(pb.WakuMessage)
		err := proto.Unmarshal(message.Data, msg)
		if err != nil {
			return false
		}

		msgHash := MsgHash(topic, msg)
		signature := msg.Meta

		return ecdsa.VerifyASN1(publicKey, msgHash, signature)
	})
}

func (w *WakuRelay) SignMessage(privKey *ecdsa.PrivateKey, topic string, msg *pb.WakuMessage) error {
	msgHash := MsgHash(topic, msg)
	sign, err := ecdsa.SignASN1(rand.Reader, privKey, msgHash)
	if err != nil {
		return err
	}

	msg.Meta = sign
	return nil
}
