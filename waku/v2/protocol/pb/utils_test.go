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

	expected := []byte{82, 136, 166, 250, 14, 69, 211, 99, 19, 161, 139, 206, 179, 3, 117, 51, 112, 111, 203, 150, 207, 35, 104, 102, 21, 181, 114, 165, 77, 29, 190, 61}
	result, err := msg.Hash()
	require.NoError(t, err)
	require.Equal(t, expected, result)
}
