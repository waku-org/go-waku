package publish

import (
	"context"
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/lightpush"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"go.uber.org/zap"
)

const peersToPublishForLightpush = 2

type PublishMethod int

const (
	LightPush PublishMethod = iota
	Relay
)

func (pm PublishMethod) String() string {
	switch pm {
	case LightPush:
		return "LightPush"
	case Relay:
		return "Relay"
	default:
		return "Unknown"
	}
}

type MessageSender struct {
	ctx              context.Context
	publishMethod    PublishMethod
	lightPush        *lightpush.WakuLightPush
	relay            *relay.WakuRelay
	messageSentCheck *MessageSentCheck
	rateLimiter      *PublishRateLimiter
	logger           *zap.Logger
}

func NewMessageSender(ctx context.Context, publishMethod PublishMethod, lightPush *lightpush.WakuLightPush, relay *relay.WakuRelay, logger *zap.Logger) *MessageSender {
	return &MessageSender{
		ctx:           ctx,
		publishMethod: publishMethod,
		lightPush:     lightPush,
		relay:         relay,
		logger:        logger,
	}
}

func (ms *MessageSender) WithMessageSentCheck(messageSentCheck *MessageSentCheck) *MessageSender {
	ms.messageSentCheck = messageSentCheck
	return ms
}

func (ms *MessageSender) WithRateLimiting(rateLimiter *PublishRateLimiter) *MessageSender {
	ms.rateLimiter = rateLimiter
	return ms
}

func (ms *MessageSender) Send(env *protocol.Envelope) error {
	logger := ms.logger.With(zap.Stringer("envelopeHash", env.Hash()), zap.String("pubsubTopic", env.PubsubTopic()), zap.String("contentTopic", env.Message().ContentTopic), zap.Int64("timestamp", env.Message().GetTimestamp()))
	if ms.rateLimiter != nil {
		if err := ms.rateLimiter.Check(ms.ctx, logger); err != nil {
			return err
		}
	}

	switch ms.publishMethod {
	case LightPush:
		if ms.lightPush == nil {
			return errors.New("lightpush is not available")
		}
		logger.Info("publishing message via lightpush")
		_, err := ms.lightPush.Publish(ms.ctx, env.Message(), lightpush.WithPubSubTopic(env.PubsubTopic()), lightpush.WithMaxPeers(peersToPublishForLightpush))
		return err
	case Relay:
		if ms.relay == nil {
			return errors.New("relay is not available")
		}
		peerCnt := len(ms.relay.PubSub().ListPeers(env.PubsubTopic()))
		logger.Info("publishing message via relay", zap.Int("peerCnt", peerCnt))
		_, err := ms.relay.Publish(ms.ctx, env.Message(), relay.WithPubSubTopic(env.PubsubTopic()))
		return err
	}

	ephemeral := env.Message().Ephemeral
	if ms.messageSentCheck != nil && (ephemeral == nil || !*ephemeral) {
		ms.messageSentCheck.Add(env.PubsubTopic(), common.BytesToHash(env.Hash().Bytes()), uint32(env.Message().GetTimestamp()/int64(time.Second)))
	}

	return nil
}

func (ms *MessageSender) PublishMethod() PublishMethod {
	return ms.publishMethod
}
