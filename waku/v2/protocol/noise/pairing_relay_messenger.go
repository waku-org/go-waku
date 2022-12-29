package noise

import (
	"context"

	n "github.com/waku-org/go-noise"
	v2 "github.com/waku-org/go-waku/waku/v2"
	"github.com/waku-org/go-waku/waku/v2/protocol"
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
	envChan chan *protocol.Envelope
	msgChan chan *pb.WakuMessage
}

type NoiseWakuRelay struct {
	NoiseMessenger
	relay                         *relay.WakuRelay
	relaySub                      *relay.Subscription
	broadcaster                   v2.Broadcaster
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
		broadcaster:                   v2.NewBroadcaster(1024),
		pubsubTopic:                   topic,
		subscriptionChPerContentTopic: make(map[string][]contentTopicSubscription),
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				subs.Unsubscribe()
				wr.broadcaster.Close()
				return
			case envelope := <-subs.C:
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
		envChan: make(chan *protocol.Envelope, 1024),
		msgChan: make(chan *pb.WakuMessage, 1024),
	}

	r.broadcaster.Register(&r.pubsubTopic, sub.envChan)

	subscriptionCh := r.subscriptionChPerContentTopic[contentTopic]
	subscriptionCh = append(subscriptionCh, sub)
	r.subscriptionChPerContentTopic[contentTopic] = subscriptionCh

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case env := <-sub.envChan:
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
	r.cancel()
	for _, contentTopicSubscriptions := range r.subscriptionChPerContentTopic {
		for _, c := range contentTopicSubscriptions {
			close(c.envChan)
			close(c.msgChan)
		}
	}
}
