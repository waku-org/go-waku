package rpc

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/status-im/go-waku/waku/v2/node"
	"github.com/status-im/go-waku/waku/v2/protocol/pb"
	"go.uber.org/zap"
)

type PrivateService struct {
	node *node.WakuNode
	log  *zap.SugaredLogger
}

type SymmetricKeyReply struct {
	Key string `json:"key"`
}

type KeyPairReply struct {
	PrivateKey string `json:"privateKey"`
	PulicKey   string `json:"publicKey"`
}

type SymmetricMessageArgs struct {
	Topic   string         `json:"topic"`
	Message pb.WakuMessage `json:"message"`
	SymKey  string         `json:"symkey"`
}

type AsymmetricMessageArgs struct {
	Topic     string         `json:"topic"`
	Message   pb.WakuMessage `json:"message"`
	PublicKey string         `json:"publicKey"`
}

type SymmetricMessagesArgs struct {
	Topic  string `json:"topic"`
	SymKey string `json:"symkey"`
}

type AsymmetricMessagesArgs struct {
	Topic      string `json:"topic"`
	PrivateKey string `json:"privateKey"`
}

func NewPrivateService(node *node.WakuNode, log *zap.SugaredLogger) *PrivateService {
	return &PrivateService{
		node: node,
		log:  log.Named("private"),
	}
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

func (p *PrivateService) PostV1SymmetricMessage(req *http.Request, args *SymmetricMessageArgs, reply *SuccessReply) error {
	keyInfo := new(node.KeyInfo)
	keyInfo.Kind = node.Symmetric
	keyInfo.SymKey = []byte(args.SymKey)

	err := node.EncodeWakuMessage(&args.Message, keyInfo)
	if err != nil {
		reply.Error = err.Error()
		reply.Success = false
	} else {
		reply.Success = true
	}

	return nil
}

func (p *PrivateService) PostV1AsymmetricMessage(req *http.Request, args *AsymmetricMessageArgs, reply *SuccessReply) error {
	keyInfo := new(node.KeyInfo)
	keyInfo.Kind = node.Asymmetric
	pubKeyBytes, err := hex.DecodeString(args.PublicKey)
	if err != nil {
		return fmt.Errorf("public key cannot be decoded: %v", err)
	}

	pubKey, err := crypto.UnmarshalPubkey(pubKeyBytes)
	if err != nil {
		return fmt.Errorf("public key cannot be unmarshalled: %v", err)
	}
	keyInfo.PubKey = *pubKey

	err = node.EncodeWakuMessage(&args.Message, keyInfo)
	if err != nil {
		reply.Error = err.Error()
		reply.Success = false
	} else {
		reply.Success = true
	}

	return nil
}

func (p *PrivateService) GetV1SymmetricMessages(req *http.Request, args *SymmetricMessagesArgs, reply *MessagesReply) error {
	return fmt.Errorf("not implemented")
}

func (p *PrivateService) GetV1AsymmetricMessages(req *http.Request, args *AsymmetricMessagesArgs, reply *MessagesReply) error {
	return fmt.Errorf("not implemented")
}
