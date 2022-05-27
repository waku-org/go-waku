package utils

import (
	"crypto/ecdsa"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/libp2p/go-libp2p-core/crypto"
)

func EcdsaPubKeyToSecp256k1PublicKey(pubKey *ecdsa.PublicKey) *crypto.Secp256k1PublicKey {
	xFieldVal := &btcec.FieldVal{}
	yFieldVal := &btcec.FieldVal{}
	xFieldVal.SetByteSlice(pubKey.X.Bytes())
	yFieldVal.SetByteSlice(pubKey.Y.Bytes())
	return (*crypto.Secp256k1PublicKey)(btcec.NewPublicKey(xFieldVal, yFieldVal))
}

func EcdsaPrivKeyToSecp256k1PrivKey(privKey *ecdsa.PrivateKey) *crypto.Secp256k1PrivateKey {
	privK, _ := btcec.PrivKeyFromBytes(privKey.D.Bytes())
	return (*crypto.Secp256k1PrivateKey)(privK)
}
