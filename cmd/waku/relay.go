package main

import (
	"bytes"
	"context"
	"sync"
	"time"

	"github.com/waku-org/go-waku/logging"
	"github.com/waku-org/go-waku/waku/v2/node"
	wprotocol "github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/rendezvous"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"go.uber.org/zap"
	"golang.org/x/exp/maps"
)

var fwdMetaTag = []byte{102, 119, 100} //"fwd"

func handleRelayTopics(ctx context.Context, wg *sync.WaitGroup, wakuNode *node.WakuNode, pubSubTopicMap map[string][]string) error {
	for nodeTopic, cTopics := range pubSubTopicMap {
		nodeTopic := nodeTopic
		_, err := wakuNode.Relay().Subscribe(ctx, wprotocol.NewContentFilter(nodeTopic, cTopics...), relay.WithoutConsumer())
		if err != nil {
			return err
		}

		if len(options.Rendezvous.Nodes) != 0 {
			// Register the node in rendezvous point
			iter := rendezvous.NewRendezvousPointIterator(options.Rendezvous.Nodes)

			wg.Add(1)
			go func(nodeTopic string) {
				t := time.NewTicker(rendezvous.RegisterDefaultTTL)
				defer t.Stop()
				defer wg.Done()

				for {
					select {
					case <-ctx.Done():
						return
					case <-t.C:
						// Register in rendezvous points periodically
						wakuNode.Rendezvous().RegisterWithNamespace(ctx, nodeTopic, iter.RendezvousPoints())
					}
				}
			}(nodeTopic)

			wg.Add(1)
			go func(nodeTopic string) {
				defer wg.Done()
				desiredOutDegree := wakuNode.Relay().Params().D
				t := time.NewTicker(7 * time.Second)
				defer t.Stop()
				for {
					select {
					case <-ctx.Done():
						return
					case <-t.C:
						peerCnt := len(wakuNode.Relay().PubSub().ListPeers(nodeTopic))
						peersToFind := desiredOutDegree - peerCnt
						if peersToFind <= 0 {
							continue
						}

						rp := <-iter.Next(ctx)
						if rp == nil {
							continue
						}
						ctx, cancel := context.WithTimeout(ctx, 7*time.Second)
						wakuNode.Rendezvous().DiscoverWithNamespace(ctx, nodeTopic, rp, peersToFind)
						cancel()
					}
				}
			}(nodeTopic)

		}
	}

	// Protected topics
	for _, protectedTopic := range options.Relay.ProtectedTopics {
		if err := wakuNode.Relay().AddSignedTopicValidator(protectedTopic.Topic, protectedTopic.PublicKey); err != nil {
			return nonRecoverErrorMsg("could not add signed topic validator: %w", err)
		}
	}

	err := bridgeTopics(ctx, wg, wakuNode)
	if err != nil {
		return err
	}

	return nil
}

func bridgeTopics(ctx context.Context, wg *sync.WaitGroup, wakuNode *node.WakuNode) error {
	// Bridge topics
	bridgedTopics := make(map[string]map[string]struct{})
	bridgedTopicsSet := make(map[string]struct{})
	for _, topics := range options.Relay.BridgeTopics {
		_, ok := bridgedTopics[topics.FromTopic]
		if !ok {
			bridgedTopics[topics.FromTopic] = make(map[string]struct{})
		}

		bridgedTopics[topics.FromTopic][topics.ToTopic] = struct{}{}
		bridgedTopicsSet[topics.FromTopic] = struct{}{}
		bridgedTopicsSet[topics.ToTopic] = struct{}{}
	}

	// Make sure all topics are subscribed
	for _, topic := range maps.Keys(bridgedTopicsSet) {
		if !wakuNode.Relay().IsSubscribed(topic) {
			_, err := wakuNode.Relay().Subscribe(ctx, wprotocol.NewContentFilter(topic), relay.WithoutConsumer())
			if err != nil {
				return err
			}
		}
	}

	for fromTopic, toTopics := range bridgedTopics {
		subscriptions, err := wakuNode.Relay().Subscribe(ctx, wprotocol.NewContentFilter(fromTopic))
		if err != nil {
			return err
		}

		topics := maps.Keys(toTopics)
		for _, subscription := range subscriptions {
			wg.Add(1)
			go func(subscription *relay.Subscription, topics []string) {
				defer wg.Done()
				for env := range subscription.Ch {
					for _, topic := range topics {
						// HACK: message has been already fwded
						metaLen := len(env.Message().Meta)
						fwdTagLen := len(fwdMetaTag)
						if metaLen >= fwdTagLen && bytes.Equal(env.Message().Meta[metaLen-fwdTagLen:], fwdMetaTag) {
							continue
						}

						// HACK: We append magic numbers here, just so the pubsub message ID will change
						env.Message().Meta = append(env.Message().Meta, fwdMetaTag...)
						_, err := wakuNode.Relay().Publish(ctx, env.Message(), relay.WithPubSubTopic(topic))
						if err != nil {
							utils.Logger().Warn("could not bridge message", logging.HexBytes("hash", env.Hash()),
								zap.String("fromTopic", env.PubsubTopic()), zap.String("toTopic", topic),
								zap.String("contentTopic", env.Message().ContentTopic), zap.Error(err))
						}
					}
				}
			}(subscription, topics)
		}
	}

	return nil
}
