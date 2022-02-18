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
	msg.Timestamp = 123456789123456789
	msg.Version = 1

	expected := []byte{210, 65, 134, 59, 106, 26, 242, 81, 17, 153, 82, 253, 107, 231, 251, 228, 49, 148, 161, 104, 111, 213, 249, 89, 85, 99, 198, 9, 2, 63, 174, 236}
	result, err := msg.Hash()

	require.NoError(t, err)
	require.Equal(t, expected, result)
}
