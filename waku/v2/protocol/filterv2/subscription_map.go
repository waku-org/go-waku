package filterv2

import (
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	v2 "github.com/waku-org/go-waku/waku/v2"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/timesource"
)

var ErrNotFound = errors.New("not found")

type ContentTopicSet map[string]struct{}

type PeerSet map[peer.ID]struct{}

type PubsubTopics map[string]ContentTopicSet // pubsubTopic => contentTopics

type SubscriptionMap struct {
	sync.RWMutex
	timesource timesource.Timesource

	items       map[peer.ID]PubsubTopics
	interestMap map[string]PeerSet // key: sha256(pubsubTopic-contentTopic) => peers

	timeout     time.Duration
	failedPeers map[peer.ID]time.Time

	broadcaster v2.Broadcaster
}

type SubscriptionItem struct {
	Key   peer.ID
	Value PubsubTopics
}

func NewSubscriptionMap(broadcaster v2.Broadcaster, timesource timesource.Timesource, timeout time.Duration) *SubscriptionMap {
	return &SubscriptionMap{
		timesource:  timesource,
		items:       make(map[peer.ID]PubsubTopics),
		interestMap: make(map[string]PeerSet),
		broadcaster: broadcaster,
		timeout:     timeout,
		failedPeers: make(map[peer.ID]time.Time),
	}
}

func (sub *SubscriptionMap) Set(peerID peer.ID, pubsubTopic string, contentTopics []string) {
	sub.Lock()
	defer sub.Unlock()

	pubsubTopicMap, ok := sub.items[peerID]
	if !ok {
		pubsubTopicMap = make(PubsubTopics)
	}

	contentTopicsMap, ok := pubsubTopicMap[pubsubTopic]
	if !ok {
		contentTopicsMap = make(ContentTopicSet)
	}

	for _, c := range contentTopics {
		contentTopicsMap[c] = struct{}{}
	}

	pubsubTopicMap[pubsubTopic] = contentTopicsMap

	sub.items[peerID] = pubsubTopicMap

	if len(contentTopics) == 0 {
		// Interested in all messages for a pubsub topic
		sub.addToInterestMap(peerID, pubsubTopic, nil)
	} else {
		for _, c := range contentTopics {
			c := c
			sub.addToInterestMap(peerID, pubsubTopic, &c)
		}
	}
}

func (sub *SubscriptionMap) Get(peerID peer.ID) (PubsubTopics, bool) {
	sub.RLock()
	defer sub.RUnlock()

	value, ok := sub.items[peerID]

	return value, ok
}

func (sub *SubscriptionMap) Has(peerID peer.ID) bool {
	sub.RLock()
	defer sub.RUnlock()

	_, ok := sub.items[peerID]

	return ok
}

func (sub *SubscriptionMap) Delete(peerID peer.ID, pubsubTopic string, contentTopics []string) error {
	sub.Lock()
	defer sub.Unlock()

	pubsubTopicMap, ok := sub.items[peerID]
	if !ok {
		return ErrNotFound
	}

	contentTopicsMap, ok := pubsubTopicMap[pubsubTopic]
	if !ok {
		return ErrNotFound
	}

	if len(contentTopics) == 0 {
		// Remove all content topics related to this pubsub topic
		for c := range contentTopicsMap {
			c := c
			delete(contentTopicsMap, c)
			sub.removeFromInterestMap(peerID, pubsubTopic, &c)
		}

		delete(pubsubTopicMap, pubsubTopic)
		sub.removeFromInterestMap(peerID, pubsubTopic, nil)
	} else {
		// Removing content topics individually
		for _, c := range contentTopics {
			c := c
			delete(contentTopicsMap, c)
			sub.removeFromInterestMap(peerID, pubsubTopic, &c)
		}

		// No more content topics available. Removing subscription completely
		if len(contentTopicsMap) == 0 {
			delete(pubsubTopicMap, pubsubTopic)
			sub.removeFromInterestMap(peerID, pubsubTopic, nil)
		}
	}

	pubsubTopicMap[pubsubTopic] = contentTopicsMap
	sub.items[peerID] = pubsubTopicMap

	return nil
}

func (sub *SubscriptionMap) deleteAll(peerID peer.ID) error {
	pubsubTopicMap, ok := sub.items[peerID]
	if !ok {
		return ErrNotFound
	}

	for pubsubTopic, contentTopicsMap := range pubsubTopicMap {
		// Remove all content topics related to this pubsub topic
		for c := range contentTopicsMap {
			c := c
			delete(contentTopicsMap, c)
			sub.removeFromInterestMap(peerID, pubsubTopic, &c)
		}

		delete(pubsubTopicMap, pubsubTopic)
		sub.removeFromInterestMap(peerID, pubsubTopic, nil)
	}

	delete(sub.items, peerID)

	return nil
}

func (sub *SubscriptionMap) DeleteAll(peerID peer.ID) error {
	sub.Lock()
	defer sub.Unlock()

	return sub.deleteAll(peerID)
}

func (sub *SubscriptionMap) RemoveAll() {
	sub.Lock()
	defer sub.Unlock()

	for k /*, _ v*/ := range sub.items {
		//close(v.Chan)
		delete(sub.items, k)
	}
}
func (sub *SubscriptionMap) Items(pubsubTopic string, contentTopic string) <-chan peer.ID {
	c := make(chan peer.ID)

	onlyPubsubTopicKey := getKey(pubsubTopic, nil)
	pubsubAndContentTopicKey := getKey(pubsubTopic, &contentTopic)

	f := func() {
		sub.RLock()
		defer sub.RUnlock()
		for p := range sub.interestMap[onlyPubsubTopicKey] {
			c <- p
		}
		for p := range sub.interestMap[pubsubAndContentTopicKey] {
			c <- p
		}
		close(c)
	}
	go f()

	return c
}

func (fm *SubscriptionMap) Notify(msg *pb.WakuMessage, peerID peer.ID) {
	/*fm.RLock()
	defer fm.RUnlock()

	filter, ok := fm.items[peerID]
	if !ok {
		return
	}

	envelope := protocol.NewEnvelope(msg, fm.timesource.Now().UnixNano(), filter.Topic)

	// Broadcasting message so it's stored
	fm.broadcaster.Submit(envelope)

	if msg.ContentTopic == "" {
		filter.Chan <- envelope
	}

	// TODO: In case of no topics we should either trigger here for all messages,
	// or we should not allow such filter to exist in the first place.
	for _, contentTopic := range filter.ContentFilters {
		if msg.ContentTopic == contentTopic {
			filter.Chan <- envelope
			break
		}
	}*/
}

func (sub *SubscriptionMap) addToInterestMap(peerID peer.ID, pubsubTopic string, contentTopic *string) {
	key := getKey(pubsubTopic, contentTopic)
	peerSet, ok := sub.interestMap[key]
	if !ok {
		peerSet = make(PeerSet)
	}
	peerSet[peerID] = struct{}{}
	sub.interestMap[key] = peerSet
}

func (sub *SubscriptionMap) removeFromInterestMap(peerID peer.ID, pubsubTopic string, contentTopic *string) {
	key := getKey(pubsubTopic, contentTopic)
	delete(sub.interestMap, key)
}

func getKey(pubsubTopic string, contentTopic *string) string {
	pubsubTopicBytes := []byte(pubsubTopic)
	if contentTopic == nil {
		return hex.EncodeToString(crypto.Keccak256(pubsubTopicBytes))
	} else {
		key := append(pubsubTopicBytes, []byte(*contentTopic)...)
		return hex.EncodeToString(crypto.Keccak256(key))
	}
}

func (sub *SubscriptionMap) IsFailedPeer(peerID peer.ID) bool {
	sub.RLock()
	defer sub.RUnlock()
	_, ok := sub.failedPeers[peerID]
	return ok
}

func (sub *SubscriptionMap) FlagAsSuccess(peerID peer.ID) {
	sub.Lock()
	defer sub.Unlock()

	_, ok := sub.failedPeers[peerID]
	if ok {
		delete(sub.failedPeers, peerID)
	}
}

func (sub *SubscriptionMap) FlagAsFailure(peerID peer.ID) {
	sub.Lock()
	defer sub.Unlock()

	lastFailure, ok := sub.failedPeers[peerID]
	if ok {
		elapsedTime := time.Since(lastFailure)
		if elapsedTime > sub.timeout {
			sub.deleteAll(peerID)
		}
	} else {
		sub.failedPeers[peerID] = time.Now()
	}
}
