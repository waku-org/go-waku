package filter

import (
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const TOPIC = "/test/topic"

func createPeerId(t *testing.T) peer.ID {
	peerId, err := test.RandPeerID()
	assert.NoError(t, err)
	return peerId
}

func firstSubscriber(subs *SubscribersMap, pubsubTopic string, contentTopic string) peer.ID {
	for sub := range subs.Items(pubsubTopic, contentTopic) {
		return sub
	}
	return ""
}

func TestAppend(t *testing.T) {
	subs := NewSubscribersMap(5 * time.Second)
	peerId := createPeerId(t)

	subs.Set(peerId, TOPIC, []string{"topic1"})

	sub := firstSubscriber(subs, TOPIC, "topic1")
	assert.NotEmpty(t, sub)

	// Adding to existing peer
	subs.Set(peerId, TOPIC, []string{"topic2"})

	sub = firstSubscriber(subs, TOPIC, "topic2")
	assert.NotEmpty(t, sub)

	subs.Set(peerId, TOPIC+"2", []string{"topic1"})

	sub = firstSubscriber(subs, TOPIC+"2", "topic1")
	assert.NotEmpty(t, sub)

	sub = firstSubscriber(subs, TOPIC+"2", "topic2")
	assert.Empty(t, sub)
}

func TestRemove(t *testing.T) {
	subs := NewSubscribersMap(5 * time.Second)
	peerId := createPeerId(t)

	subs.Set(peerId, TOPIC+"1", []string{"topic1", "topic2"})
	subs.Set(peerId, TOPIC+"2", []string{"topic1"})

	err := subs.DeleteAll(peerId)
	assert.Empty(t, err)

	sub := firstSubscriber(subs, TOPIC+"1", "topic1")
	assert.Empty(t, sub)

	sub = firstSubscriber(subs, TOPIC+"1", "topic2")
	assert.Empty(t, sub)

	sub = firstSubscriber(subs, TOPIC+"2", "topic1")
	assert.Empty(t, sub)

	assert.False(t, subs.Has(peerId))

	_, found := subs.Get(peerId)
	assert.False(t, found)

	_, ok := subs.items[peerId]
	assert.False(t, ok)
}

func TestRemovePartial(t *testing.T) {
	subs := NewSubscribersMap(5 * time.Second)
	peerId := createPeerId(t)

	subs.Set(peerId, TOPIC, []string{"topic1", "topic2"})
	err := subs.Delete(peerId, TOPIC, []string{"topic1"})
	require.NoError(t, err)

	sub := firstSubscriber(subs, TOPIC, "topic2")
	assert.NotEmpty(t, sub)
}

func TestRemoveBogus(t *testing.T) {
	subs := NewSubscribersMap(5 * time.Second)
	peerId := createPeerId(t)

	subs.Set(peerId, TOPIC, []string{"topic1", "topic2"})
	err := subs.Delete(peerId, TOPIC, []string{"does not exist", "topic1"})
	require.NoError(t, err)

	sub := firstSubscriber(subs, TOPIC, "topic1")
	assert.Empty(t, sub)
	sub = firstSubscriber(subs, TOPIC, "does not exist")
	assert.Empty(t, sub)

	err = subs.Delete(peerId, "DOES_NOT_EXIST", []string{"topic1"})
	require.Error(t, err)
}

func TestSuccessFailure(t *testing.T) {
	subs := NewSubscribersMap(5 * time.Second)
	peerId := createPeerId(t)

	subs.Set(peerId, TOPIC, []string{"topic1", "topic2"})

	subs.FlagAsFailure(peerId)
	require.True(t, subs.IsFailedPeer(peerId))

	subs.FlagAsFailure(peerId)
	require.False(t, subs.Has(peerId))

	subs.Set(peerId, TOPIC, []string{"topic1", "topic2"})

	subs.FlagAsFailure(peerId)
	require.True(t, subs.IsFailedPeer(peerId))

	subs.FlagAsSuccess(peerId)
	require.False(t, subs.IsFailedPeer(peerId))
}
