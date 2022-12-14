package noise

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/require"
	n "github.com/waku-org/noise"
)

func TestWakuPairing(t *testing.T) {
	// Pairing Phase
	// ==========

	// Alice static/ephemeral key initialization and commitment
	aliceStaticKey, _ := n.DH25519.GenerateKeypair(rand.Reader)
	aliceEphemeralKey, _ := n.DH25519.GenerateKeypair(rand.Reader)
	s := generateRandomBytes(t, 32)
	aliceCommittedStaticKey := CommitPublicKey(aliceStaticKey.Public, s)

	// Bob static/ephemeral key initialization and commitment
	bobStaticKey, _ := n.DH25519.GenerateKeypair(rand.Reader)
	bobEphemeralKey, _ := n.DH25519.GenerateKeypair(rand.Reader)
	r := generateRandomBytes(t, 32)
	bobCommittedStaticKey := CommitPublicKey(bobStaticKey.Public, r)

	// Content topic information
	applicationName := "waku-noise-sessions"
	applicationVersion := "0.1"
	shardId := "10"
	qrMessageNameTag := BytesToMessageNametag(generateRandomBytes(t, MessageNametagLength))

	// Out-of-band Communication

	// Bob prepares the QR and sends it out-of-band to Alice
	qr := NewQR(applicationName, applicationVersion, shardId, bobEphemeralKey.Public, bobCommittedStaticKey)

	// Alice deserializes the QR code
	readQR, err := StringToQR(qr.String())
	require.NoError(t, err)

	// We check if QR serialization/deserialization works
	require.Equal(t, applicationName, readQR.applicationName)
	require.Equal(t, applicationVersion, readQR.applicationVersion)
	require.Equal(t, shardId, readQR.shardId)
	require.True(t, bytes.Equal(bobEphemeralKey.Public, readQR.ephemeralKey))
	require.True(t, bytes.Equal(bobCommittedStaticKey[:], readQR.committedStaticKey[:]))

	// Pre-handshake message
	// <- eB {H(sB||r), contentTopicParams, messageNametag}
	preMessagePKs := bobEphemeralKey.Public

	// We initialize the Handshake states.
	// Note that we pass the whole qr serialization as prologue information

	aliceHS, err := NewHandshake_WakuPairing_25519_ChaChaPoly_SHA256(aliceStaticKey, aliceEphemeralKey, true, qr.Bytes(), preMessagePKs)
	require.NoError(t, err)

	bobHS, err := NewHandshake_WakuPairing_25519_ChaChaPoly_SHA256(bobStaticKey, bobEphemeralKey, false, qr.Bytes(), preMessagePKs)
	require.NoError(t, err)

	// Pairing Handshake
	// ==========

	// Write and read calls alternate between Alice and Bob: the handhshake progresses by alternatively calling stepHandshake for each user

	// 1st step
	// -> eA, eAeB   {H(sA||s)}   [authcode]

	// The messageNametag for the first handshake message is randomly generated and exchanged out-of-band
	// and corresponds to qrMessageNametag

	// We set the transport message to be H(sA||s)
	sentTransportMessage := aliceCommittedStaticKey

	// By being the handshake initiator, Alice writes a Waku2 payload v2 containing her handshake message
	// and the (encrypted) transport message
	// The message is sent with a messageNametag equal to the one received through the QR code
	aliceStep, err := aliceHS.Step(nil, sentTransportMessage, &qrMessageNameTag)
	require.NoError(t, err)

	// Bob reads Alice's payloads, and returns the (decrypted) transport message Alice sent to him
	// Note that Bob verifies if the received payloadv2 has the expected messageNametag set
	bobStep, err := bobHS.Step(&aliceStep.Payload2, nil, &qrMessageNameTag)
	require.NoError(t, err)

	require.True(t, bytes.Equal(bobStep.TransportMessage, sentTransportMessage))

	// We generate an authorization code using the handshake state
	aliceAuthcode, err := aliceHS.Authcode()
	require.NoError(t, err)

	bobAuthcode, err := bobHS.Authcode()
	require.NoError(t, err)

	// We check that they are equal. Note that this check has to be confirmed with a user interaction.
	require.Equal(t, aliceAuthcode, bobAuthcode)

	// 2nd step
	// <- sB, eAsB    {r}

	// Alice and Bob update their local next messageNametag using the available handshake information
	// During the handshake, messageNametag = HKDF(h), where h is the handshake hash value at the end of the last processed message
	aliceMessageNametag, err := aliceHS.ToMessageNametag()
	require.NoError(t, err)

	bobMessageNametag, err := bobHS.ToMessageNametag()
	require.NoError(t, err)

	// We set as a transport message the commitment randomness r
	sentTransportMessage = r

	// At this step, Bob writes and returns a payload
	bobStep, err = bobHS.Step(nil, sentTransportMessage, &bobMessageNametag)
	require.NoError(t, err)

	// While Alice reads and returns the (decrypted) transport message
	aliceStep, err = aliceHS.Step(&bobStep.Payload2, nil, &aliceMessageNametag)
	require.NoError(t, err)
	require.Equal(t, aliceStep.TransportMessage, sentTransportMessage)

	// Alice further checks if Bob's commitment opens to Bob's static key she just received
	expectedBobCommittedStaticKey := CommitPublicKey(aliceHS.RS(), aliceStep.TransportMessage)
	require.True(t, bytes.Equal(expectedBobCommittedStaticKey, bobCommittedStaticKey))

	// 3rd step
	// -> sA, sAeB, sAsB  {s}

	// Alice and Bob update their local next messageNametag using the available handshake information
	aliceMessageNametag, err = aliceHS.ToMessageNametag()
	require.NoError(t, err)

	bobMessageNametag, err = bobHS.ToMessageNametag()
	require.NoError(t, err)

	// We set as a transport message the commitment randomness s
	sentTransportMessage = s

	// Similarly as in first step, Alice writes a Waku2 payload containing the handshake message and the (encrypted) transport message
	aliceStep, err = aliceHS.Step(nil, sentTransportMessage, &aliceMessageNametag)
	require.NoError(t, err)

	// Bob reads Alice's payloads, and returns the (decrypted) transport message Alice sent to him
	bobStep, err = bobHS.Step(&aliceStep.Payload2, nil, &bobMessageNametag)
	require.NoError(t, err)
	require.True(t, bytes.Equal(bobStep.TransportMessage, sentTransportMessage))

	// Bob further checks if Alice's commitment opens to Alice's static key he just received
	expectedAliceCommittedStaticKey := CommitPublicKey(bobHS.RS(), bobStep.TransportMessage)

	require.True(t, bytes.Equal(expectedAliceCommittedStaticKey, aliceCommittedStaticKey))

	// Secure Transfer Phase
	// ==========

	// We test read/write of random messages exchanged between Alice and Bob
	// Note that we exchange more than the number of messages contained in the nametag buffer to test if they are filled correctly as the communication proceeds
	for i := 0; i < 10*MessageNametagBufferSize; i++ {
		// Alice writes to Bob
		message := generateRandomBytes(t, 32)
		payload, err := aliceHS.Encrypt(message)
		require.NoError(t, err)

		readMessage, err := bobHS.Decrypt(payload)
		require.NoError(t, err)
		require.True(t, bytes.Equal(message, readMessage))

		// Bob writes to Alice
		message = generateRandomBytes(t, 32)
		payload, err = bobHS.Encrypt(message)
		require.NoError(t, err)

		readMessage, err = aliceHS.Decrypt(payload)
		require.NoError(t, err)
		require.True(t, bytes.Equal(message, readMessage))
	}

	// We test how nametag buffers help in detecting lost messages
	// Alice writes two messages to Bob, but only the second is received
	message := generateRandomBytes(t, 32)
	_, err = aliceHS.Encrypt(message)
	require.NoError(t, err)

	message = generateRandomBytes(t, 32)
	payload2, err := aliceHS.Encrypt(message)
	require.NoError(t, err)

	_, err = bobHS.Decrypt(payload2)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNametagNotExpected)

	// We adjust bob nametag buffer for next test (i.e. the missed message is correctly recovered)
	bobHS.nametagsInbound.Delete(2)
	message = generateRandomBytes(t, 32)
	payload2, err = bobHS.Encrypt(message)
	require.NoError(t, err)
	readMessage, err := aliceHS.Decrypt(payload2)
	require.NoError(t, err)
	require.True(t, bytes.Equal(message, readMessage))

	// We test if a missing nametag is correctly detected
	message = generateRandomBytes(t, 32)
	payload2, err = aliceHS.Encrypt(message)
	require.NoError(t, err)
	bobHS.nametagsInbound.Delete(1)
	_, err = bobHS.Decrypt(payload2)
	require.ErrorIs(t, err, ErrNametagNotFound)
}
