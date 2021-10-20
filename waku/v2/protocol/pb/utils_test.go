package pb

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHash(t *testing.T) {
	expected := []byte{89, 47, 167, 67, 136, 159, 199, 249, 42, 194, 163, 123, 177, 245, 186, 29, 175, 42, 92, 132, 116, 28, 160, 224, 6, 29, 36, 58, 46, 103, 7, 186}
	result := Hash([]byte("Hello World"))
	require.Equal(t, expected, result)
}

func TestEnvelopeHash(t *testing.T) {
	msg := new(WakuMessage)
	msg.ContentTopic = "Test"
	msg.Payload = []byte("Hello World")
	msg.Timestamp = float64(123456789123456789)
	msg.Version = 1

	expected := []byte{77, 197, 250, 41, 30, 163, 192, 239, 48, 104, 58, 175, 36, 81, 96, 58, 118, 107, 73, 4, 153, 182, 33, 199, 144, 156, 110, 226, 93, 85, 160, 180}
	result, err := msg.Hash()

	require.NoError(t, err)
	require.Equal(t, expected, result)
}
