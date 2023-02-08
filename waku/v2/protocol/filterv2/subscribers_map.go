package filterv2

import (
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
)

var ErrNotFound = errors.New("not found")

type ContentTopicSet map[string]struct{}

type PeerSet map[peer.ID]struct{}

type PubsubTopics map[string]ContentTopicSet // pubsubTopic => contentTopics

type SubscribersMap struct {
	sync.RWMutex

	items       map[peer.ID]PubsubTopics
	interestMap map[string]PeerSet // key: sha256(pubsubTopic-contentTopic) => peers

	timeout     time.Duration
	failedPeers map[peer.ID]time.Time
}

func NewSubscribersMap(timeout time.Duration) *SubscribersMap {
	return &SubscribersMap{
		items:       make(map[peer.ID]PubsubTopics),
		interestMap: make(map[string]PeerSet),
		timeout:     timeout,
		failedPeers: make(map[peer.ID]time.Time),
	}
}

func (sub *SubscribersMap) Set(peerID peer.ID, pubsubTopic string, contentTopics []string) {
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

func (sub *SubscribersMap) Get(peerID peer.ID) (PubsubTopics, bool) {
	sub.RLock()
	defer sub.RUnlock()

	value, ok := sub.items[peerID]

	return value, ok
}

func (sub *SubscribersMap) Has(peerID peer.ID) bool {
	sub.RLock()
	defer sub.RUnlock()

	_, ok := sub.items[peerID]

	return ok
}

func (sub *SubscribersMap) Delete(peerID peer.ID, pubsubTopic string, contentTopics []string) error {
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

func (sub *SubscribersMap) deleteAll(peerID peer.ID) error {
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

func (sub *SubscribersMap) DeleteAll(peerID peer.ID) error {
	sub.Lock()
	defer sub.Unlock()

	return sub.deleteAll(peerID)
}

func (sub *SubscribersMap) RemoveAll() {
	sub.Lock()
	defer sub.Unlock()

	for k /*, _ v*/ := range sub.items {
		//close(v.Chan)
		delete(sub.items, k)
	}
}

func (sub *SubscribersMap) Items(pubsubTopic string, contentTopic string) <-chan peer.ID {
	c := make(chan peer.ID)

	onlyPubsubTopicKey := getKey(pubsubTopic, nil)
	pubsubAndContentTopicKey := getKey(pubsubTopic, &contentTopic)

	f := func() {
		sub.RLock()
		defer sub.RUnlock()

		if peers, ok := sub.interestMap[onlyPubsubTopicKey]; ok {
			for p := range peers {
				c <- p
			}
		}

		if peers, ok := sub.interestMap[pubsubAndContentTopicKey]; ok {
			for p := range peers {
				c <- p
			}
		}
		close(c)
	}
	go f()

	return c
}

func (sub *SubscribersMap) addToInterestMap(peerID peer.ID, pubsubTopic string, contentTopic *string) {
	key := getKey(pubsubTopic, contentTopic)
	peerSet, ok := sub.interestMap[key]
	if !ok {
		peerSet = make(PeerSet)
	}
	peerSet[peerID] = struct{}{}
	sub.interestMap[key] = peerSet
}

func (sub *SubscribersMap) removeFromInterestMap(peerID peer.ID, pubsubTopic string, contentTopic *string) {
	key := getKey(pubsubTopic, contentTopic)
	_, exists := sub.interestMap[key]
	if exists {
		delete(sub.interestMap[key], peerID)
	}
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

func (sub *SubscribersMap) IsFailedPeer(peerID peer.ID) bool {
	sub.RLock()
	defer sub.RUnlock()
	_, ok := sub.failedPeers[peerID]
	return ok
}

func (sub *SubscribersMap) FlagAsSuccess(peerID peer.ID) {
	sub.Lock()
	defer sub.Unlock()

	_, ok := sub.failedPeers[peerID]
	if ok {
		delete(sub.failedPeers, peerID)
	}
}

func (sub *SubscribersMap) FlagAsFailure(peerID peer.ID) {
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
