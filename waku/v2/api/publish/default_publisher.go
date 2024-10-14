package publish

import (
	"context"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/waku-org/go-waku/waku/v2/protocol/lightpush"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
)

func NewDefaultPublisher(lightpush *lightpush.WakuLightPush, relay *relay.WakuRelay) Publisher {
	return &defaultPublisher{
		lightPush: lightpush,
		relay:     relay,
	}
}

type defaultPublisher struct {
	lightPush *lightpush.WakuLightPush
	relay     *relay.WakuRelay
}

func (d *defaultPublisher) RelayListPeers(pubsubTopic string) []peer.ID {
	return d.relay.PubSub().ListPeers(pubsubTopic)
}

func (d *defaultPublisher) RelayPublish(ctx context.Context, message *pb.WakuMessage, pubsubTopic string) (pb.MessageHash, error) {
	return d.relay.Publish(ctx, message, relay.WithPubSubTopic(pubsubTopic))
}

func (d *defaultPublisher) LightpushPublish(ctx context.Context, message *pb.WakuMessage, pubsubTopic string, maxPeers int) (pb.MessageHash, error) {
	return d.lightPush.Publish(ctx, message, lightpush.WithPubSubTopic(pubsubTopic), lightpush.WithMaxPeers(maxPeers))
}
