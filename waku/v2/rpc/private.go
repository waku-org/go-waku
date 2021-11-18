package rpc

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/status-im/go-waku/waku/v2/node"
)

type PrivateService struct {
	node *node.WakuNode
}

type SymmetricKeyReply struct {
	Key string `json:"key"`
}

type KeyPairReply struct {
	PrivateKey string `json:"privateKey"`
	PulicKey   string `json:"publicKey"`
}

func (p *PrivateService) GetV1SymmetricKey(req *http.Request, args *Empty, reply *SymmetricKeyReply) error {
	key := [32]byte{}
	_, err := rand.Read(key[:])
	if err != nil {
		return err
	}
	reply.Key = hex.EncodeToString(key[:])
	return nil
}

func (p *PrivateService) GetV1AsymmetricKeypair(req *http.Request, args *Empty, reply *KeyPairReply) error {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		return err
	}
	privateKeyBytes := crypto.FromECDSA(privateKey)

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return fmt.Errorf("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
	}

	publicKeyBytes := crypto.FromECDSAPub(publicKeyECDSA)
	reply.PrivateKey = hex.EncodeToString(privateKeyBytes[:])
	reply.PulicKey = hex.EncodeToString(publicKeyBytes[:])
	return nil
}

func (p *PrivateService) PostV1SymmetricMessage(req *http.Request, args *FilterContentFilterArgs, reply *SuccessReply) error {
	return fmt.Errorf("not implemented")
}

func (p *PrivateService) PostV1AsymmetricMessage(req *http.Request, args *FilterContentFilterArgs, reply *SuccessReply) error {
	return fmt.Errorf("not implemented")
}

func (p *PrivateService) GetV1SymmetricMessages(req *http.Request, args *FilterContentFilterArgs, reply *SuccessReply) error {
	return fmt.Errorf("not implemented")
}

func (p *PrivateService) GetV1AsymmetricMessages(req *http.Request, args *FilterContentFilterArgs, reply *SuccessReply) error {
	return fmt.Errorf("not implemented")
}
