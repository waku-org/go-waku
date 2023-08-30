package protocol

import (
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestContentTopicAndSharding(t *testing.T) {
	ct, err := NewContentTopic("waku", 2, "test", "proto")
	require.NoError(t, err)
	require.Equal(t, ct.String(), "/waku/2/test/proto")

	_, err = StringToContentTopic("/waku/-1/a/b")
	require.Error(t, ErrInvalidFormat, err)

	_, err = StringToContentTopic("waku/1/a/b")
	require.Error(t, ErrInvalidFormat, err)

	_, err = StringToContentTopic("////")
	require.Error(t, ErrInvalidFormat, err)

	_, err = StringToContentTopic("/waku/1/a")
	require.Error(t, ErrInvalidFormat, err)

	ct2, err := StringToContentTopic("/waku/2/test/proto")
	require.NoError(t, err)
	require.Equal(t, ct.String(), ct2.String())
	require.True(t, ct.Equal(ct2))

	ct3, err := NewContentTopic("waku", 2, "test2", "proto")
	require.NoError(t, err)
	require.False(t, ct.Equal(ct3))

	ct4, err := StringToContentTopic("/0/toychat/2/huilong/proto")
	require.NoError(t, err)
	require.Equal(t, ct4.Generation, 0)

	ct6, err := StringToContentTopic("/toychat/2/huilong/proto")
	require.NoError(t, err)

	nsPubSubT1 := GetShardFromContentTopic(ct6, GenerationZeroShardsCount)
	require.Equal(t, NewStaticShardingPubsubTopic(ClusterIndex, 3), nsPubSubT1)

	_, err = StringToContentTopic("/abc/toychat/2/huilong/proto")
	require.Error(t, ErrInvalidGeneration, err)

	_, err = StringToContentTopic("/1/toychat/2/huilong/proto")
	require.Error(t, ErrInvalidGeneration, err)

	ct5, err := NewContentTopic("waku", 2, "test2", "proto", WithGeneration(0))
	require.NoError(t, err)
	require.Equal(t, ct5.Generation, 0)
}

func randomContentTopic() (ContentTopic, error) {
	var app = ""
	const WordLength = 5
	rand.New(rand.NewSource(time.Now().Unix()))

	//Generate a random character between lowercase a to z
	for i := 0; i < WordLength; i++ {
		randomChar := 'a' + rune(rand.Intn(26))
		app = app + string(randomChar)
	}
	version := uint32(1)

	var name = ""

	for i := 0; i < WordLength; i++ {
		randomChar := 'a' + rune(rand.Intn(26))
		name = name + string(randomChar)
	}
	var enc = "proto"

	return NewContentTopic(app, version, name, enc)
}

func TestShardChoiceSimulation(t *testing.T) {
	//Given
	var topics []ContentTopic
	for i := 0; i < 100000; i++ {
		ct, err := randomContentTopic()
		require.NoError(t, err)
		topics = append(topics, ct)
	}

	var counts [GenerationZeroShardsCount]int

	// When
	for _, topic := range topics {
		pubsub := GetShardFromContentTopic(topic, GenerationZeroShardsCount)
		counts[pubsub.Shard()] += 1
	}

	for i := 0; i < GenerationZeroShardsCount; i++ {
		t.Logf("Counts at index %d is %d", i, counts[i])
	}

	// Then
	for i := 1; i < GenerationZeroShardsCount; i++ {
		if float64(counts[i-1]) <= (float64(counts[i])*1.05) &&
			float64(counts[i]) <= (float64(counts[i-1])*1.05) &&
			float64(counts[i-1]) >= (float64(counts[i])*0.95) &&
			float64(counts[i]) >= (float64(counts[i-1])*0.95) {
			t.Logf("Shard choice simulation successful")
		} else {
			t.FailNow()
		}
	}
}

func TestNsPubsubTopic(t *testing.T) {
	ns1 := NewNamedShardingPubsubTopic("waku-dev")
	require.Equal(t, "/waku/2/waku-dev", ns1.String())

	ns2 := NewStaticShardingPubsubTopic(0, 2)
	require.Equal(t, "/waku/2/rs/0/2", ns2.String())

	require.True(t, ns1.Equal(ns1))
	require.False(t, ns1.Equal(ns2))

	topic := "/waku/2/waku-dev"
	ns, err := ToShardedPubsubTopic(topic)
	require.NoError(t, err)
	require.Equal(t, NamedSharding, ns.Kind())
	require.Equal(t, "waku-dev", ns.(NamedShardingPubsubTopic).Name())

	topic = "/waku/2/rs/16/42"
	ns, err = ToShardedPubsubTopic(topic)
	require.NoError(t, err)
	require.Equal(t, StaticSharding, ns.Kind())
	require.Equal(t, uint16(16), ns.(StaticShardingPubsubTopic).Cluster())
	require.Equal(t, uint16(42), ns.(StaticShardingPubsubTopic).Shard())

	topic = "/waku/1/rs/16/42"
	_, err = ToShardedPubsubTopic(topic)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrInvalidTopicPrefix)

	topic = "/waku/2/rs//02"
	_, err = ToShardedPubsubTopic(topic)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrMissingClusterIndex)

	topic = "/waku/2/rs/xx/77"
	_, err = ToShardedPubsubTopic(topic)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrInvalidNumberFormat)
}
