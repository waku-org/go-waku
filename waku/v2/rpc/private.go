package rpc

import (
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/status-im/go-waku/waku/v2/node"
	"github.com/status-im/go-waku/waku/v2/protocol"
	"github.com/status-im/go-waku/waku/v2/protocol/pb"
	"github.com/status-im/go-waku/waku/v2/protocol/relay"
	"go.uber.org/zap"
)

type PrivateService struct {
	node *node.WakuNode
	log  *zap.Logger

	messages      map[string][]*pb.WakuMessage
	messagesMutex sync.RWMutex

	runner *runnerService
}

type SymmetricKeyReply string

type KeyPairReply struct {
	PrivateKey string `json:"privateKey"`
	PublicKey  string `json:"publicKey"`
}

type SymmetricMessageArgs struct {
	Topic   string         `json:"topic"`
	Message RPCWakuMessage `json:"message"`
	SymKey  string         `json:"symkey"`
}

type AsymmetricMessageArgs struct {
	Topic     string         `json:"topic"`
	Message   RPCWakuMessage `json:"message"`
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

func NewPrivateService(node *node.WakuNode, log *zap.Logger) *PrivateService {
	p := &PrivateService{
		node:     node,
		messages: make(map[string][]*pb.WakuMessage),
		log:      log.Named("private"),
	}
	p.runner = newRunnerService(node.Broadcaster(), p.addEnvelope)

	return p
}

func (p *PrivateService) addEnvelope(envelope *protocol.Envelope) {
	p.messagesMutex.Lock()
	defer p.messagesMutex.Unlock()
	if _, ok := p.messages[envelope.PubsubTopic()]; !ok {
		p.messages[envelope.PubsubTopic()] = make([]*pb.WakuMessage, 0)
	}

	p.messages[envelope.PubsubTopic()] = append(p.messages[envelope.PubsubTopic()], envelope.Message())
}

func (p *PrivateService) GetV1SymmetricKey(req *http.Request, args *Empty, reply *SymmetricKeyReply) error {
	key := [32]byte{}
	_, err := rand.Read(key[:])
	if err != nil {
		return err
	}
	*reply = SymmetricKeyReply(hexutil.Encode(key[:]))
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
	reply.PrivateKey = hexutil.Encode(privateKeyBytes[:])
	reply.PublicKey = hexutil.Encode(publicKeyBytes[:])
	return nil
}

func (p *PrivateService) PostV1SymmetricMessage(req *http.Request, args *SymmetricMessageArgs, reply *SuccessReply) error {
	symKeyBytes, err := hexutil.Decode(args.SymKey)
	if err != nil {
		return fmt.Errorf("invalid symmetric key: %w", err)
	}

	keyInfo := new(node.KeyInfo)
	keyInfo.Kind = node.Symmetric
	keyInfo.SymKey = symKeyBytes

	msg := args.Message.toProto()
	msg.Version = 1

	err = node.EncodeWakuMessage(msg, keyInfo)
	if err != nil {
		reply.Error = err.Error()
		reply.Success = false
		return nil
	}

	topic := args.Topic
	if topic == "" {
		topic = relay.DefaultWakuTopic
	}

	_, err = p.node.Relay().PublishToTopic(req.Context(), msg, topic)
	if err != nil {
		reply.Error = err.Error()
		reply.Success = false
		return nil
	}

	reply.Success = true
	return nil
}

func (p *PrivateService) PostV1AsymmetricMessage(req *http.Request, args *AsymmetricMessageArgs, reply *bool) error {
	keyInfo := new(node.KeyInfo)
	keyInfo.Kind = node.Asymmetric

	pubKeyBytes, err := hexutil.Decode(args.PublicKey)
	if err != nil {
		return fmt.Errorf("public key cannot be decoded: %v", err)
	}

	pubKey, err := crypto.UnmarshalPubkey(pubKeyBytes)
	if err != nil {
		return fmt.Errorf("public key cannot be unmarshalled: %v", err)
	}

	keyInfo.PubKey = *pubKey

	msg := args.Message.toProto()
	msg.Version = 1

	err = node.EncodeWakuMessage(msg, keyInfo)
	if err != nil {
		return err
	}

	topic := args.Topic
	if topic == "" {
		topic = relay.DefaultWakuTopic
	}

	_, err = p.node.Relay().PublishToTopic(req.Context(), msg, topic)
	if err != nil {
		return err
	}

	*reply = true

	return nil
}

func (p *PrivateService) GetV1SymmetricMessages(req *http.Request, args *SymmetricMessagesArgs, reply *MessagesReply) error {
	p.messagesMutex.Lock()
	defer p.messagesMutex.Unlock()

	if _, ok := p.messages[args.Topic]; !ok {
		p.messages[args.Topic] = make([]*pb.WakuMessage, 0)
	}

	symKeyBytes, err := hexutil.Decode(args.SymKey)
	if err != nil {
		return fmt.Errorf("invalid symmetric key: %w", err)
	}

	messages := make([]*pb.WakuMessage, len(p.messages[args.Topic]))
	copy(messages, p.messages[args.Topic])
	p.messages[args.Topic] = make([]*pb.WakuMessage, 0)

	var decodedMessages []*pb.WakuMessage
	for _, msg := range messages {
		err := node.DecodeWakuMessage(msg, &node.KeyInfo{
			Kind:   node.Symmetric,
			SymKey: symKeyBytes,
		})
		if err != nil {
			continue
		}
		decodedMessages = append(decodedMessages, msg)
	}

	for i := range decodedMessages {
		*reply = append(*reply, ProtoWakuMessageToRPCWakuMessage(decodedMessages[i]))
	}

	return nil
}

func (p *PrivateService) GetV1AsymmetricMessages(req *http.Request, args *AsymmetricMessagesArgs, reply *MessagesReply) error {
	p.messagesMutex.Lock()
	defer p.messagesMutex.Unlock()

	if _, ok := p.messages[args.Topic]; !ok {
		p.messages[args.Topic] = make([]*pb.WakuMessage, 0)
	}

	messages := make([]*pb.WakuMessage, len(p.messages[args.Topic]))
	copy(messages, p.messages[args.Topic])
	p.messages[args.Topic] = make([]*pb.WakuMessage, 0)

	privKey, err := crypto.HexToECDSA(strings.TrimPrefix(args.PrivateKey, "0x"))
	if err != nil {
		return fmt.Errorf("invalid asymmetric key: %w", err)
	}

	var decodedMessages []*pb.WakuMessage
	for _, msg := range messages {
		err := node.DecodeWakuMessage(msg, &node.KeyInfo{
			Kind:    node.Asymmetric,
			PrivKey: privKey,
		})
		if err != nil {
			continue
		}
		decodedMessages = append(decodedMessages, msg)
	}

	for i := range decodedMessages {
		*reply = append(*reply, ProtoWakuMessageToRPCWakuMessage(decodedMessages[i]))
	}

	return nil
}

func (p *PrivateService) Start() {
	p.runner.Start()
}

func (p *PrivateService) Stop() {
	p.runner.Stop()
}
