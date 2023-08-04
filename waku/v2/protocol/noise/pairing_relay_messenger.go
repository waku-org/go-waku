package noise

import (
	"context"

	n "github.com/waku-org/go-noise"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/timesource"
)

type NoiseMessenger interface {
	Sender
	Receiver
	Stop()
}

type contentTopicSubscription struct {
	broadcastSub relay.Subscription
	msgChan      chan *pb.WakuMessage
}

type NoiseWakuRelay struct {
	NoiseMessenger
	relay                         *relay.WakuRelay
	relaySub                      *relay.Subscription
	broadcaster                   relay.Broadcaster
	cancel                        context.CancelFunc
	timesource                    timesource.Timesource
	pubsubTopic                   string
	subscriptionChPerContentTopic map[string][]contentTopicSubscription
}

func NewWakuRelayMessenger(ctx context.Context, r *relay.WakuRelay, pubsubTopic *string, timesource timesource.Timesource) (*NoiseWakuRelay, error) {
	var topic string
	if pubsubTopic != nil {
		topic = *pubsubTopic
	} else {
		topic = relay.DefaultWakuTopic
	}

	subs, err := r.SubscribeToTopic(ctx, topic)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(ctx)

	wr := &NoiseWakuRelay{
		relay:                         r,
		relaySub:                      subs,
		cancel:                        cancel,
		timesource:                    timesource,
		broadcaster:                   relay.NewBroadcaster(1024),
		pubsubTopic:                   topic,
		subscriptionChPerContentTopic: make(map[string][]contentTopicSubscription),
	}

	err = wr.broadcaster.Start(ctx)
	if err != nil {
		return nil, err
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				subs.Unsubscribe()
				wr.broadcaster.Stop()
				return
			case envelope := <-subs.Ch:
				if envelope != nil {
					wr.broadcaster.Submit(envelope)
				}
			}
		}
	}()

	return wr, nil
}

func (r *NoiseWakuRelay) Subscribe(ctx context.Context, contentTopic string) <-chan *pb.WakuMessage {
	sub := contentTopicSubscription{
		msgChan: make(chan *pb.WakuMessage, 1024),
	}

	broadcastSub := r.broadcaster.RegisterForAll(1024)
	sub.broadcastSub = broadcastSub

	subscriptionCh := r.subscriptionChPerContentTopic[contentTopic]
	subscriptionCh = append(subscriptionCh, sub)
	r.subscriptionChPerContentTopic[contentTopic] = subscriptionCh

	go func() {
		for {
			select {
			case <-ctx.Done():
				close(sub.msgChan)
				return
			case env := <-sub.broadcastSub.Ch:
				if env == nil {
					return
				}

				if env.Message().ContentTopic != contentTopic || env.Message().Version != 2 {
					continue
				}

				// TODO: Might make sense to create a ring buffer here, to drop messages if queue fills up
				sub.msgChan <- env.Message()
			}
		}
	}()

	return sub.msgChan
}

func (r *NoiseWakuRelay) Publish(ctx context.Context, contentTopic string, payload *n.PayloadV2) error {

	message, err := EncodePayloadV2(payload)
	if err != nil {
		return err
	}

	message.ContentTopic = contentTopic
	message.Timestamp = r.timesource.Now().UnixNano()

	_, err = r.relay.PublishToTopic(ctx, message, r.pubsubTopic)
	return err
}

func (r *NoiseWakuRelay) Stop() {
	if r.cancel == nil {
		return
	}

	r.cancel()
	for _, contentTopicSubscriptions := range r.subscriptionChPerContentTopic {
		for _, c := range contentTopicSubscriptions {
			c.broadcastSub.Unsubscribe()
		}
	}
}
