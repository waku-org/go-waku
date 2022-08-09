package noise

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"

	"github.com/flynn/noise"
	"github.com/stretchr/testify/require"
)

func generateRandomBytes(t *testing.T, n int) []byte {
	b := make([]byte, n)
	_, err := rand.Read(b)
	require.NoError(t, err)
	return b
}

func TestSerialization(t *testing.T) {
	handshakeMessages := make([]*NoisePublicKey, 2)

	pk1, _, _ := ed25519.GenerateKey(rand.Reader)

	pk2, _, _ := ed25519.GenerateKey(rand.Reader)

	handshakeMessages[0] = Ed25519PubKeyToNoisePublicKey(pk1)
	handshakeMessages[1] = Ed25519PubKeyToNoisePublicKey(pk2)

	p1 := &PayloadV2{
		ProtocolId:       Noise_K1K1_25519_ChaChaPoly_SHA256,
		HandshakeMessage: handshakeMessages,
		TransportMessage: []byte{9, 8, 7, 6, 5, 4, 3, 2, 1},
	}

	serializedPayload, err := p1.Serialize()
	require.NoError(t, err)

	deserializedPayload, err := DeserializePayloadV2(serializedPayload)
	require.NoError(t, err)
	require.Equal(t, p1, deserializedPayload)
}

func handshakeTest(t *testing.T, hsAlice *Handshake, hsBob *Handshake) {
	// ###############
	// # 1st step
	// ###############

	// By being the handshake initiator, Alice writes a Waku2 payload v2 containing her handshake message
	// and the (encrypted) transport message
	sentTransportMessage := generateRandomBytes(t, 32)
	aliceStep, err := hsAlice.Step(nil, sentTransportMessage)
	require.NoError(t, err)

	// Bob reads Alice's payloads, and returns the (decrypted) transport message Alice sent to him
	bobStep, err := hsBob.Step(&aliceStep.Payload2, nil)
	require.NoError(t, err)

	// check:
	require.Equal(t, sentTransportMessage, bobStep.TransportMessage)

	// ###############
	// # 2nd step
	// ###############

	// At this step, Bob writes and returns a payload
	sentTransportMessage = generateRandomBytes(t, 32)
	bobStep, err = hsBob.Step(nil, sentTransportMessage)
	require.NoError(t, err)

	// While Alice reads and returns the (decrypted) transport message
	aliceStep, err = hsAlice.Step(&bobStep.Payload2, nil)
	require.NoError(t, err)

	// check:
	require.Equal(t, sentTransportMessage, aliceStep.TransportMessage)

	// ###############
	// # 3rd step
	// ###############

	// Similarly as in first step, Alice writes a Waku2 payload containing the handshake message and the (encrypted) transport message
	sentTransportMessage = generateRandomBytes(t, 32)
	aliceStep, err = hsAlice.Step(nil, sentTransportMessage)
	require.NoError(t, err)

	// Bob reads Alice's payloads, and returns the (decrypted) transport message Alice sent to him
	bobStep, err = hsBob.Step(&aliceStep.Payload2, nil)
	require.NoError(t, err)

	// check:
	require.Equal(t, sentTransportMessage, bobStep.TransportMessage)

	// Note that for this handshake pattern, no more message patterns are left for processing
	// We test that extra calls to stepHandshake do not affect parties' handshake states
	require.True(t, hsAlice.HandshakeComplete())
	require.True(t, hsBob.HandshakeComplete())

	_, err = hsAlice.Step(nil, generateRandomBytes(t, 32))
	require.ErrorIs(t, err, ErrorHandshakeComplete)

	_, err = hsBob.Step(nil, generateRandomBytes(t, 32))
	require.ErrorIs(t, err, ErrorHandshakeComplete)

	// #########################
	// After Handshake
	// #########################

	// We test read/write of random messages exchanged between Alice and Bob

	for i := 0; i < 10; i++ {
		// Alice writes to Bob
		message := generateRandomBytes(t, 32)

		encryptedPayload, err := hsAlice.Encrypt(message)
		require.NoError(t, err)

		plaintext, err := hsBob.Decrypt(encryptedPayload)
		require.NoError(t, err)

		require.Equal(t, message, plaintext)

		// Bob writes to Alice
		message = generateRandomBytes(t, 32)

		encryptedPayload, err = hsBob.Encrypt(message)
		require.NoError(t, err)

		plaintext, err = hsAlice.Decrypt(encryptedPayload)
		require.NoError(t, err)

		require.Equal(t, message, plaintext)
	}
}

func TestNoiseXXHandshakeRoundtrip(t *testing.T) {
	aliceKP, _ := noise.DH25519.GenerateKeypair(rand.Reader)
	bobKP, _ := noise.DH25519.GenerateKeypair(rand.Reader)

	hsAlice, err := NewHandshake_XX_25519_ChaChaPoly_SHA256(aliceKP, true, nil)
	require.NoError(t, err)

	hsBob, err := NewHandshake_XX_25519_ChaChaPoly_SHA256(bobKP, false, nil)
	require.NoError(t, err)

	handshakeTest(t, hsAlice, hsBob)
}

func TestNoiseXXpsk0HandshakeRoundtrip(t *testing.T) {
	aliceKP, _ := noise.DH25519.GenerateKeypair(rand.Reader)
	bobKP, _ := noise.DH25519.GenerateKeypair(rand.Reader)

	// We generate a random psk
	psk := generateRandomBytes(t, 32)

	hsAlice, err := NewHandshake_XXpsk0_25519_ChaChaPoly_SHA256(aliceKP, true, psk, nil)
	require.NoError(t, err)

	hsBob, err := NewHandshake_XXpsk0_25519_ChaChaPoly_SHA256(bobKP, false, psk, nil)
	require.NoError(t, err)

	handshakeTest(t, hsAlice, hsBob)
}

func TestNoiseK1K1HandshakeRoundtrip(t *testing.T) {
	aliceKP, _ := noise.DH25519.GenerateKeypair(rand.Reader)
	bobKP, _ := noise.DH25519.GenerateKeypair(rand.Reader)

	hsAlice, err := NewHandshake_K1K1_25519_ChaChaPoly_SHA256(aliceKP, true, bobKP.Public, nil)
	require.NoError(t, err)

	hsBob, err := NewHandshake_K1K1_25519_ChaChaPoly_SHA256(bobKP, false, aliceKP.Public, nil)
	require.NoError(t, err)

	handshakeTest(t, hsAlice, hsBob)
}

func TestNoiseXK1HandshakeRoundtrip(t *testing.T) {
	aliceKP, _ := noise.DH25519.GenerateKeypair(rand.Reader)
	bobKP, _ := noise.DH25519.GenerateKeypair(rand.Reader)

	hsAlice, err := NewHandshake_XK1_25519_ChaChaPoly_SHA256(aliceKP, true, bobKP.Public, nil)
	require.NoError(t, err)

	hsBob, err := NewHandshake_XK1_25519_ChaChaPoly_SHA256(bobKP, false, nil, nil)
	require.NoError(t, err)

	handshakeTest(t, hsAlice, hsBob)
}
