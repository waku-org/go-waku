package node

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	crand "crypto/rand"
	mrand "math/rand"

	"errors"
	"strconv"

	"github.com/ethereum/go-ethereum/crypto/ecies"
	"github.com/status-im/go-waku/waku/v2/protocol"
)

type KeyKind string

const (
	Symmetric  KeyKind = "Symmetric"
	Asymmetric KeyKind = "Asymmetric"
	None       KeyKind = "None"
)

type KeyInfo struct {
	kind    KeyKind
	symKey  []byte
	privKey ecdsa.PrivateKey
}

// NOTICE: Extracted from status-go

const aesNonceLength = 12
const aesKeyLength = 32

// Decrypts a message with a topic key, using AES-GCM-256.
// nonce size should be 12 bytes (see cipher.gcmStandardNonceSize).
func decryptSymmetric(payload []byte, key []byte) ([]byte, error) {
	// symmetric messages are expected to contain the 12-byte nonce at the end of the payload
	if len(payload) < aesNonceLength {
		return nil, errors.New("missing salt or invalid payload in symmetric message")
	}

	salt := payload[len(payload)-aesNonceLength:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	decrypted, err := aesgcm.Open(nil, salt, payload[:len(payload)-aesNonceLength], nil)
	if err != nil {
		return nil, err
	}

	return decrypted, nil
}

// Decrypts an encrypted payload with a private key.
func decryptAsymmetric(payload []byte, key *ecdsa.PrivateKey) ([]byte, error) {
	decrypted, err := ecies.ImportECDSA(key).Decrypt(payload, nil, nil)
	if err == nil {
		return nil, err
	}
	return decrypted, err
}

func decodePayload(message *protocol.WakuMessage, keyInfo *KeyInfo) ([]byte, error) {
	switch *message.Version {
	case uint32(0):
		return message.Payload, nil
	case uint32(1):
		switch keyInfo.kind {
		case Symmetric:
			decoded, err := decryptSymmetric(message.Payload, keyInfo.symKey)
			if err != nil {
				return nil, errors.New("Couldn't decrypt using symmetric key")
			} else {
				return decoded, nil
			}
		case Asymmetric:
			decoded, err := decryptAsymmetric(message.Payload, &keyInfo.privKey)
			if err != nil {
				return nil, errors.New("Couldn't decrypt using asymmetric key")
			} else {
				return decoded, nil
			}
		case None:
			return nil, errors.New("Non supported KeyKind")
		}
	}
	return nil, errors.New("Unsupported WakuMessage version")
}

// ValidatePublicKey checks the format of the given public key.
func ValidatePublicKey(k *ecdsa.PublicKey) bool {
	return k != nil && k.X != nil && k.Y != nil && k.X.Sign() != 0 && k.Y.Sign() != 0
}

// Encrypts and returns with a public key.
func encryptAsymmetric(rawPayload []byte, key *ecdsa.PublicKey) ([]byte, error) {
	if !ValidatePublicKey(key) {
		return nil, errors.New("invalid public key provided for asymmetric encryption")
	}
	encrypted, err := ecies.Encrypt(crand.Reader, ecies.ImportECDSAPublic(key), rawPayload, nil, nil)
	if err == nil {
		return encrypted, nil
	}
	return nil, err
}

// Encrypts a payload with a topic key, using AES-GCM-256.
// nonce size should be 12 bytes (see cipher.gcmStandardNonceSize).
func encryptSymmetric(rawPayload []byte, key []byte) ([]byte, error) {
	if !validateDataIntegrity(key, aesKeyLength) {
		return nil, errors.New("invalid key provided for symmetric encryption, size: " + strconv.Itoa(len(key)))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	salt, err := generateSecureRandomData(aesNonceLength) // never use more than 2^32 random nonces with a given key
	if err != nil {
		return nil, err
	}
	encrypted := aesgcm.Seal(nil, salt, rawPayload, nil)
	return append(encrypted, salt...), nil
}

// validateDataIntegrity returns false if the data have the wrong or contains all zeros,
// which is the simplest and the most common bug.
func validateDataIntegrity(k []byte, expectedSize int) bool {
	if len(k) != expectedSize {
		return false
	}
	if expectedSize > 3 && containsOnlyZeros(k) {
		return false
	}
	return true
}

// containsOnlyZeros checks if the data contain only zeros.
func containsOnlyZeros(data []byte) bool {
	for _, b := range data {
		if b != 0 {
			return false
		}
	}
	return true
}

// generateSecureRandomData generates random data where extra security is required.
// The purpose of this function is to prevent some bugs in software or in hardware
// from delivering not-very-random data. This is especially useful for AES nonce,
// where true randomness does not really matter, but it is very important to have
// a unique nonce for every message.
func generateSecureRandomData(length int) ([]byte, error) {
	x := make([]byte, length)
	y := make([]byte, length)
	res := make([]byte, length)

	_, err := crand.Read(x)
	if err != nil {
		return nil, err
	} else if !validateDataIntegrity(x, length) {
		return nil, errors.New("crypto/rand failed to generate secure random data")
	}
	_, err = mrand.Read(y)
	if err != nil {
		return nil, err
	} else if !validateDataIntegrity(y, length) {
		return nil, errors.New("math/rand failed to generate secure random data")
	}
	for i := 0; i < length; i++ {
		res[i] = x[i] ^ y[i]
	}
	if !validateDataIntegrity(res, length) {
		return nil, errors.New("failed to generate secure random data")
	}
	return res, nil
}

func encode(rawPayload []byte, keyInfo *KeyInfo, version uint32) ([]byte, error) {
	switch version {
	case 0:
		return rawPayload, nil
	case 1:
		switch keyInfo.kind {
		case Symmetric:
			encoded, err := encryptSymmetric(rawPayload, keyInfo.symKey)
			if err != nil {
				return nil, errors.New("Couldn't encrypt using symmetric key")
			} else {
				return encoded, nil
			}
		case Asymmetric:
			encoded, err := encryptAsymmetric(rawPayload, &keyInfo.privKey.PublicKey)
			if err != nil {
				return nil, errors.New("Couldn't encrypt using asymmetric key")
			} else {
				return encoded, nil
			}
		case None:
			return nil, errors.New("Non supported KeyKind")
		}
	}
	return nil, errors.New("Unsupported WakuMessage version")
}
