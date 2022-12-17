package noise

import (
	"bytes"
	"errors"
	"fmt"
	"hash"
	"io"
	"math/big"

	n "github.com/waku-org/noise"
	"golang.org/x/crypto/hkdf"
)

// WakuNoiseProtocolID indicates the protocol ID defined according to https://rfc.vac.dev/spec/35/#specification
type WakuNoiseProtocolID = byte

var (
	None                                      = WakuNoiseProtocolID(0)
	Noise_K1K1_25519_ChaChaPoly_SHA256        = WakuNoiseProtocolID(10)
	Noise_XK1_25519_ChaChaPoly_SHA256         = WakuNoiseProtocolID(11)
	Noise_XX_25519_ChaChaPoly_SHA256          = WakuNoiseProtocolID(12)
	Noise_XXpsk0_25519_ChaChaPoly_SHA256      = WakuNoiseProtocolID(13)
	Noise_WakuPairing_25519_ChaChaPoly_SHA256 = WakuNoiseProtocolID(14)
	ChaChaPoly                                = WakuNoiseProtocolID(30)
)

const NoisePaddingBlockSize = 248

var ErrorHandshakeComplete = errors.New("handshake complete")

// All protocols share same cipher suite
var cipherSuite = n.NewCipherSuite(n.DH25519, n.CipherChaChaPoly, n.HashSHA256)

func newHandshakeState(pattern n.HandshakePattern, initiator bool, staticKeypair n.DHKey, ephemeralKeyPair n.DHKey, prologue []byte, presharedKey []byte, peerStatic []byte, peerEphemeral []byte) (hs *n.HandshakeState, err error) {
	defer func() {
		if rerr := recover(); rerr != nil {
			err = fmt.Errorf("panic in Noise handshake: %s", rerr)
		}
	}()

	cfg := n.Config{
		CipherSuite:      cipherSuite,
		Pattern:          pattern,
		Initiator:        initiator,
		StaticKeypair:    staticKeypair,
		EphemeralKeypair: ephemeralKeyPair,
		Prologue:         prologue,
		PresharedKey:     presharedKey,
		PeerStatic:       peerStatic,
		PeerEphemeral:    peerEphemeral,
	}

	return n.NewHandshakeState(cfg)
}

type Handshake struct {
	protocolID WakuNoiseProtocolID
	pattern    n.HandshakePattern
	state      *n.HandshakeState

	hsBuff []byte

	enc              *n.CipherState
	dec              *n.CipherState
	nametagsInbound  *MessageNametagBuffer
	nametagsOutbound *MessageNametagBuffer

	initiator   bool
	shouldWrite bool
}

// HandshakeStepResult stores the intermediate result of processing messages patterns
type HandshakeStepResult struct {
	Payload2         PayloadV2
	TransportMessage []byte
}

func getHandshakePattern(protocol WakuNoiseProtocolID) (n.HandshakePattern, error) {
	switch protocol {
	case Noise_K1K1_25519_ChaChaPoly_SHA256:
		return HandshakeK1K1, nil
	case Noise_XK1_25519_ChaChaPoly_SHA256:
		return HandshakeXK1, nil
	case Noise_XX_25519_ChaChaPoly_SHA256:
		return HandshakeXX, nil
	case Noise_XXpsk0_25519_ChaChaPoly_SHA256:
		return HandshakeXXpsk0, nil
	case Noise_WakuPairing_25519_ChaChaPoly_SHA256:
		return HandshakeWakuPairing, nil
	default:
		return n.HandshakePattern{}, errors.New("unsupported handshake pattern")
	}
}

// NewHandshake creates a new handshake using aa WakuNoiseProtocolID that is maped to a handshake pattern.
func NewHandshake(protocolID WakuNoiseProtocolID, initiator bool, staticKeypair n.DHKey, ephemeralKeyPair n.DHKey, prologue []byte, presharedKey []byte, peerStatic []byte, peerEphemeral []byte) (*Handshake, error) {
	hsPattern, err := getHandshakePattern(protocolID)
	if err != nil {
		return nil, err
	}

	hsState, err := newHandshakeState(hsPattern, initiator, staticKeypair, ephemeralKeyPair, prologue, presharedKey, peerStatic, peerEphemeral)
	if err != nil {
		return nil, err
	}

	return &Handshake{
		protocolID:  protocolID,
		pattern:     hsPattern,
		initiator:   initiator,
		shouldWrite: initiator,
		state:       hsState,
	}, nil
}

func (hs *Handshake) Hash() hash.Hash {
	return hs.state.Hash()
}

func (hs *Handshake) H() []byte {
	return hs.state.H()
}

func (hs *Handshake) RS() []byte {
	return hs.state.RS()
}

// Step advances a step in the handshake. Each user in a handshake alternates writing and reading of handshake messages.
// If the user is writing the handshake message, the transport message (if not empty) has to be passed to transportMessage and readPayloadV2 can be left to its default value
// It the user is reading the handshake message, the read payload v2 has to be passed to readPayloadV2 and the transportMessage can be left to its default values.
// TODO: this might be refactored into a separate `sendHandshakeMessage` and `receiveHandshakeMessage`.
// The messageNameTag is an optional value and can be used to identify missing messages
func (hs *Handshake) Step(readPayloadV2 *PayloadV2, transportMessage []byte, messageNametag *MessageNametag) (*HandshakeStepResult, error) {
	if hs.enc != nil || hs.dec != nil {
		return nil, ErrorHandshakeComplete
	}

	var cs1 *n.CipherState
	var cs2 *n.CipherState
	var err error
	var msg []byte
	var noisePubKeys [][]byte

	result := HandshakeStepResult{}

	if hs.shouldWrite {
		// We initialize a payload v2 and we set proper protocol ID (if supported)
		result.Payload2.ProtocolId = hs.protocolID

		payload, err := PKCS7_Pad(transportMessage, NoisePaddingBlockSize)
		if err != nil {
			return nil, err
		}

		var mtag MessageNametag
		if messageNametag != nil {
			mtag = *messageNametag
		}

		msg, noisePubKeys, cs1, cs2, err = hs.state.WriteMessageAndGetPK(hs.hsBuff, [][]byte{}, payload, mtag[:])
		if err != nil {
			return nil, err
		}

		hs.shouldWrite = false

		if messageNametag != nil {
			result.Payload2.MessageNametag = *messageNametag
		}

		result.Payload2.TransportMessage = msg
		for _, npk := range noisePubKeys {
			result.Payload2.HandshakeMessage = append(result.Payload2.HandshakeMessage, byteToNoisePublicKey(npk))
		}

	} else {
		if readPayloadV2 == nil {
			return nil, errors.New("readPayloadV2 is required")
		}

		var mtag MessageNametag
		if messageNametag != nil {
			mtag = *messageNametag
			if !bytes.Equal(readPayloadV2.MessageNametag[:], mtag[:]) {
				return nil, ErrNametagNotExpected
			}
		}

		readTMessage := readPayloadV2.TransportMessage

		// We retrieve and store the (decrypted) received transport message by passing the messageNametag as extra additional data

		// Since we only read, nothing meanigful (i.e. public keys) is returned. (hsBuffer is not affected)
		msg, cs1, cs2, err = hs.state.ReadMessage(nil, readTMessage, mtag[:]...)
		if err != nil {
			return nil, err
		}

		hs.shouldWrite = true

		// We retrieve, and store the (unpadded decrypted) received transport message

		payload, err := PKCS7_Unpad(msg, NoisePaddingBlockSize)
		if err != nil {
			return nil, err
		}

		result.TransportMessage = payload
	}

	if cs1 != nil && cs2 != nil {
		err = hs.setCipherStates(cs1, cs2)
		if err != nil {
			return nil, err
		}
	}

	return &result, nil
}

// HandshakeComplete indicates whether the handshake process is complete or not
func (hs *Handshake) HandshakeComplete() bool {
	return hs.enc != nil && hs.dec != nil
}

// This is called when the final handshake message is processed
func (hs *Handshake) setCipherStates(cs1, cs2 *n.CipherState) error {
	// Optional: We derive a secret for the nametag derivation
	nms1, nms2, err := hs.messageNametagSecrets()
	if err != nil {
		return err
	}

	if hs.initiator {
		hs.enc = cs1
		hs.dec = cs2
		// and nametags secrets
		hs.nametagsInbound = NewMessageNametagBuffer(nms1)
		hs.nametagsOutbound = NewMessageNametagBuffer(nms2)
	} else {
		hs.enc = cs2
		hs.dec = cs1
		// and nametags secrets
		hs.nametagsInbound = NewMessageNametagBuffer(nms2)
		hs.nametagsOutbound = NewMessageNametagBuffer(nms1)
	}

	// We initialize the message nametags inbound/outbound buffers
	hs.nametagsInbound.Init()
	hs.nametagsOutbound.Init()

	return nil
}

// Encrypt calls the cipher's encryption. It encrypts the provided plaintext and returns a PayloadV2
func (hs *Handshake) Encrypt(plaintext []byte, outboundMessageNametagBuffer ...*MessageNametagBuffer) (*PayloadV2, error) {
	if hs.enc == nil {
		return nil, errors.New("cannot encrypt, handshake incomplete")
	}

	if len(plaintext) == 0 {
		return nil, errors.New("tried to encrypt empty plaintext")
	}

	paddedTransportMessage, err := PKCS7_Pad(plaintext, NoisePaddingBlockSize)
	if err != nil {
		return nil, err
	}

	// We set the message nametag using the input buffer
	var messageNametag MessageNametag
	if len(outboundMessageNametagBuffer) != 0 {
		messageNametag = outboundMessageNametagBuffer[0].Pop()
	} else {
		messageNametag = hs.nametagsOutbound.Pop()
	}

	cyphertext, err := hs.enc.Encrypt(nil, messageNametag[:], paddedTransportMessage)
	if err != nil {
		return nil, err
	}

	// According to 35/WAKU2-NOISE RFC, no Handshake protocol information is sent when exchanging messages
	// This correspond to setting protocol-id to 0 (None)
	return &PayloadV2{
		ProtocolId:       None,
		TransportMessage: cyphertext,
		MessageNametag:   messageNametag,
	}, nil
}

// Decrypt calls the cipher's decryption. It decrypts the provided payload and returns the message in plaintext
func (hs *Handshake) Decrypt(payload *PayloadV2, inboundMessageNametagBuffer ...*MessageNametagBuffer) ([]byte, error) {
	if hs.dec == nil {
		return nil, errors.New("cannot decrypt, handshake incomplete")
	}

	if payload == nil {
		return nil, errors.New("no payload to decrypt")
	}

	// If the message nametag does not correspond to the nametag expected in the inbound message nametag buffer
	// an error is raised (to be handled externally, i.e. re-request lost messages, discard, etc.)
	if len(inboundMessageNametagBuffer) != 0 {
		err := inboundMessageNametagBuffer[0].CheckNametag(payload.MessageNametag)
		if err != nil {
			return nil, err
		}
	} else {
		err := hs.nametagsInbound.CheckNametag(payload.MessageNametag)
		if err != nil {
			return nil, err
		}
	}

	if len(payload.TransportMessage) == 0 {
		return nil, errors.New("tried to decrypt empty ciphertext")
	}

	// Decryption is done with messageNametag as associated data
	paddedMessage, err := hs.dec.Decrypt(nil, payload.MessageNametag[:], payload.TransportMessage)
	if err != nil {
		return nil, err
	}

	// The message successfully decrypted, we can delete the first element of the inbound Message Nametag Buffer
	hs.nametagsInbound.Delete(1)

	return PKCS7_Unpad(paddedMessage, NoisePaddingBlockSize)
}

func getHKDF(h func() hash.Hash, ck []byte, ikm []byte, numBytes int) ([]byte, error) {
	hkdf := hkdf.New(h, ikm, ck, nil)
	result := make([]byte, numBytes)
	if _, err := io.ReadFull(hkdf, result); err != nil {
		return nil, err
	}
	return result, nil
}

// Generates an 8 decimal digits authorization code using HKDF and the handshake state
func (hs *Handshake) Authcode() (string, error) {
	output0, err := getHKDF(hs.Hash, hs.H(), nil, 8)
	if err != nil {
		return "", err
	}
	bn := new(big.Int)
	bn.SetBytes(output0)
	code := new(big.Int)
	code.Mod(bn, big.NewInt(100_000_000))
	return fmt.Sprintf("'%08s'", code.String()), nil
}

func (hs *Handshake) messageNametagSecrets() (nms1 []byte, nms2 []byte, err error) {
	output, err := getHKDF(hs.Hash, hs.H(), nil, 64)
	if err != nil {
		return nil, nil, err
	}
	nms1 = output[0:32]
	nms2 = output[32:]
	return
}

// Uses the cryptographic information stored in the input handshake state to generate a random message nametag
// In current implementation the messageNametag = HKDF(handshake hash value), but other derivation mechanisms can be implemented
func (hs *Handshake) ToMessageNametag() (MessageNametag, error) {
	output, err := getHKDF(hs.Hash, hs.H(), nil, 32)
	if err != nil {
		return [16]byte{}, err
	}
	return BytesToMessageNametag(output), nil
}

// NewHandshake_XX_25519_ChaChaPoly_SHA256 creates a handshake where the initiator and receiver are not aware of each other static keys
func NewHandshake_XX_25519_ChaChaPoly_SHA256(staticKeypair n.DHKey, initiator bool, prologue []byte) (*Handshake, error) {
	return NewHandshake(Noise_XX_25519_ChaChaPoly_SHA256, initiator, staticKeypair, n.DHKey{}, prologue, nil, nil, nil)
}

// NewHandshake_XXpsk0_25519_ChaChaPoly_SHA256 creates a handshake where the initiator and receiver are not aware of each other static keys
// and use a preshared secret to strengthen their mutual authentication
func NewHandshake_XXpsk0_25519_ChaChaPoly_SHA256(staticKeypair n.DHKey, initiator bool, presharedKey []byte, prologue []byte) (*Handshake, error) {
	return NewHandshake(Noise_XXpsk0_25519_ChaChaPoly_SHA256, initiator, staticKeypair, n.DHKey{}, prologue, presharedKey, nil, nil)
}

// NewHandshake_K1K1_25519_ChaChaPoly_SHA256 creates a handshake where both initiator and recever know each other handshake. Only ephemeral keys
// are exchanged. This handshake is useful in case the initiator needs to instantiate a new separate encrypted communication
// channel with the receiver
func NewHandshake_K1K1_25519_ChaChaPoly_SHA256(staticKeypair n.DHKey, initiator bool, peerStaticKey []byte, prologue []byte) (*Handshake, error) {
	return NewHandshake(Noise_K1K1_25519_ChaChaPoly_SHA256, initiator, staticKeypair, n.DHKey{}, prologue, nil, peerStaticKey, nil)
}

// NewHandshake_XK1_25519_ChaChaPoly_SHA256 creates a handshake where the initiator knows the receiver public static key. Within this handshake,
// the initiator and receiver reciprocally authenticate their static keys using ephemeral keys. We note that while the receiver's
// static key is assumed to be known to Alice (and hence is not transmitted), The initiator static key is sent to the
// receiver encrypted with a key derived from both parties ephemeral keys and the receiver's static key.
func NewHandshake_XK1_25519_ChaChaPoly_SHA256(staticKeypair n.DHKey, initiator bool, peerStaticKey []byte, prologue []byte) (*Handshake, error) {
	if !initiator && len(peerStaticKey) != 0 {
		return nil, errors.New("recipient shouldnt know initiator key")
	}
	return NewHandshake(Noise_XK1_25519_ChaChaPoly_SHA256, initiator, staticKeypair, n.DHKey{}, prologue, nil, peerStaticKey, nil)
}

// NewHandshake_WakuPairing_25519_ChaChaPoly_SHA256
func NewHandshake_WakuPairing_25519_ChaChaPoly_SHA256(staticKeypair n.DHKey, ephemeralKeyPair n.DHKey, initiator bool, prologue []byte, presharedKey []byte) (*Handshake, error) {
	peerEphemeral := presharedKey[0:32]
	return NewHandshake(Noise_WakuPairing_25519_ChaChaPoly_SHA256, initiator, staticKeypair, ephemeralKeyPair, prologue, presharedKey, nil, peerEphemeral)
}
