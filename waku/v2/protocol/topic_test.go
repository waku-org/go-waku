package protocol

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContentTopic(t *testing.T) {
	ct := NewContentTopic("waku", 2, "test", "proto")
	require.Equal(t, ct.String(), "/waku/2/test/proto")

	_, err := StringToContentTopic("/waku/-1/a/b")
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

	ct3 := NewContentTopic("waku", 2, "test2", "proto")
	require.False(t, ct.Equal(ct3))
}

func TestTopic(t *testing.T) {
	topic := NewPubsubTopic("test", "proto")
	require.Equal(t, topic.String(), "/waku/2/test/proto")

	_, err := StringToPubsubTopic("/waku/-1/a/b")
	require.Error(t, ErrInvalidFormat, err)

	_, err = StringToPubsubTopic("waku/2/a/b")
	require.Error(t, ErrInvalidFormat, err)

	_, err = StringToPubsubTopic("////")
	require.Error(t, ErrInvalidFormat, err)

	_, err = StringToPubsubTopic("/waku/2/a")
	require.Error(t, ErrInvalidFormat, err)

	topic2, err := StringToPubsubTopic("/waku/2/test/proto")
	require.NoError(t, err)
	require.Equal(t, topic.String(), topic2.String())
	require.True(t, topic.Equal(topic2))

	topic3 := NewPubsubTopic("test2", "proto")
	require.False(t, topic.Equal(topic3))
}
