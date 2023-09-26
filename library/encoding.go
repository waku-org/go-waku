package library

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/json"
	"errors"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/waku-org/go-waku/waku/v2/payload"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

func wakuMessage(messageJSON string) (*pb.WakuMessage, error) {
	var msg *pb.WakuMessage
	err := json.Unmarshal([]byte(messageJSON), &msg)
	return msg, err
}

func EncodeSymmetric(messageJSON string, symmetricKey string, optionalSigningKey string) (*pb.WakuMessage, error) {
	msg, err := wakuMessage(messageJSON)
	if err != nil {
		return nil, err
	}

	payload := payload.Payload{
		Data: msg.Payload,
		Key: &payload.KeyInfo{
			Kind: payload.Symmetric,
		},
	}

	keyBytes, err := utils.DecodeHexString(symmetricKey)
	if err != nil {
		return nil, err
	}

	payload.Key.SymKey = keyBytes

	if optionalSigningKey != "" {
		signingKeyBytes, err := utils.DecodeHexString(optionalSigningKey)
		if err != nil {
			return nil, err
		}

		payload.Key.PrivKey, err = crypto.ToECDSA(signingKeyBytes)
		if err != nil {
			return nil, err
		}
	}

	msg.Version = 1
	msg.Payload, err = payload.Encode(1)
	if err != nil {
		return nil, err
	}

	return msg, err
}

func EncodeAsymmetric(messageJSON string, publicKey string, optionalSigningKey string) (*pb.WakuMessage, error) {
	msg, err := wakuMessage(messageJSON)
	if err != nil {
		return nil, err
	}

	payload := payload.Payload{
		Data: msg.Payload,
		Key: &payload.KeyInfo{
			Kind: payload.Asymmetric,
		},
	}

	keyBytes, err := utils.DecodeHexString(publicKey)
	if err != nil {
		return nil, err
	}

	payload.Key.PubKey, err = unmarshalPubkey(keyBytes)
	if err != nil {
		return nil, err
	}

	if optionalSigningKey != "" {
		signingKeyBytes, err := utils.DecodeHexString(optionalSigningKey)
		if err != nil {
			return nil, err
		}

		payload.Key.PrivKey, err = crypto.ToECDSA(signingKeyBytes)
		if err != nil {
			return nil, err
		}
	}

	msg.Version = 1
	msg.Payload, err = payload.Encode(1)
	if err != nil {
		return nil, err
	}

	return msg, err
}

func extractPubKeyAndSignature(payload *payload.DecodedPayload) (pubkey string, signature string) {
	pkBytes := crypto.FromECDSAPub(payload.PubKey)
	if len(pkBytes) != 0 {
		pubkey = hexutil.Encode(pkBytes)
	}

	if len(payload.Signature) != 0 {
		signature = hexutil.Encode(payload.Signature)
	}

	return
}

// DecodeSymmetric decodes a waku message using a 32 bytes symmetric key. The key must be a hex encoded string with "0x" prefix
func DecodeSymmetric(messageJSON string, symmetricKey string) (any, error) {
	var msg pb.WakuMessage
	err := json.Unmarshal([]byte(messageJSON), &msg)
	if err != nil {
		return nil, err
	}

	if msg.Version == 0 {
		return struct {
			Data    []byte `json:"data"`
			Padding []byte `json:"padding"`
		}{
			Data: msg.Payload,
		}, nil
	} else if msg.Version > 1 {
		return nil, errors.New("unsupported wakumessage version")
	}

	keyInfo := &payload.KeyInfo{
		Kind: payload.Symmetric,
	}

	keyInfo.SymKey, err = utils.DecodeHexString(symmetricKey)
	if err != nil {
		return nil, err
	}

	payload, err := payload.DecodePayload(&msg, keyInfo)
	if err != nil {
		return nil, err
	}

	pubkey, signature := extractPubKeyAndSignature(payload)

	response := struct {
		PubKey    string `json:"pubkey,omitempty"`
		Signature string `json:"signature,omitempty"`
		Data      []byte `json:"data"`
		Padding   []byte `json:"padding,omitempty"`
	}{
		PubKey:    pubkey,
		Signature: signature,
		Data:      payload.Data,
		Padding:   payload.Padding,
	}

	return response, nil
}

// DecodeAsymmetric decodes a waku message using a secp256k1 private key. The key must be a hex encoded string with "0x" prefix
func DecodeAsymmetric(messageJSON string, privateKey string) (any, error) {
	var msg pb.WakuMessage
	err := json.Unmarshal([]byte(messageJSON), &msg)
	if err != nil {
		return "", err
	}

	if msg.Version == 0 {
		return struct {
			Data    []byte `json:"data"`
			Padding []byte `json:"padding"`
		}{
			Data: msg.Payload,
		}, nil
	} else if msg.Version > 1 {
		return nil, errors.New("unsupported wakumessage version")
	}

	keyInfo := &payload.KeyInfo{
		Kind: payload.Asymmetric,
	}

	keyBytes, err := utils.DecodeHexString(privateKey)
	if err != nil {
		return nil, err
	}

	keyInfo.PrivKey, err = crypto.ToECDSA(keyBytes)
	if err != nil {
		return nil, err
	}

	payload, err := payload.DecodePayload(&msg, keyInfo)
	if err != nil {
		return nil, err
	}

	pubkey, signature := extractPubKeyAndSignature(payload)

	response := struct {
		PubKey    string `json:"pubkey,omitempty"`
		Signature string `json:"signature,omitempty"`
		Data      []byte `json:"data"`
		Padding   []byte `json:"padding"`
	}{
		PubKey:    pubkey,
		Signature: signature,
		Data:      payload.Data,
		Padding:   payload.Padding,
	}

	return response, nil
}

func unmarshalPubkey(pub []byte) (ecdsa.PublicKey, error) {
	x, y := elliptic.Unmarshal(secp256k1.S256(), pub)
	if x == nil {
		return ecdsa.PublicKey{}, errors.New("invalid public key")
	}
	return ecdsa.PublicKey{Curve: secp256k1.S256(), X: x, Y: y}, nil
}
