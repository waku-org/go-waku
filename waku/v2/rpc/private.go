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
	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-waku/waku/v2/payload"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"go.uber.org/zap"
)

type PrivateService struct {
	node *node.WakuNode
	log  *zap.Logger

	messages      map[string][]*pb.WakuMessage
	cacheCapacity int
	messagesMutex sync.RWMutex

	runner *runnerService
}

type SymmetricKeyReply string

type KeyPairReply struct {
	PrivateKey string `json:"privateKey"`
	PublicKey  string `json:"publicKey"`
}

type SymmetricMessageArgs struct {
	Topic   string          `json:"topic"`
	Message *RPCWakuMessage `json:"message"`
	SymKey  string          `json:"symkey"`
}

type AsymmetricMessageArgs struct {
	Topic     string          `json:"topic"`
	Message   *RPCWakuMessage `json:"message"`
	PublicKey string          `json:"publicKey"`
}

type SymmetricMessagesArgs struct {
	Topic  string `json:"topic"`
	SymKey string `json:"symkey"`
}

type AsymmetricMessagesArgs struct {
	Topic      string `json:"topic"`
	PrivateKey string `json:"privateKey"`
}

func NewPrivateService(node *node.WakuNode, cacheCapacity int, log *zap.Logger) *PrivateService {
	p := &PrivateService{
		node:          node,
		cacheCapacity: cacheCapacity,
		messages:      make(map[string][]*pb.WakuMessage),
		log:           log.Named("private"),
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

	// Keep a specific max number of messages per topic
	if len(p.messages[envelope.PubsubTopic()]) >= p.cacheCapacity {
		p.messages[envelope.PubsubTopic()] = p.messages[envelope.PubsubTopic()][1:]
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
	symKeyBytes, err := utils.DecodeHexString(args.SymKey)
	if err != nil {
		return fmt.Errorf("invalid symmetric key: %w", err)
	}

	keyInfo := new(payload.KeyInfo)
	keyInfo.Kind = payload.Symmetric
	keyInfo.SymKey = symKeyBytes

	msg := args.Message
	msg.Version = 1

	protoMsg := msg.toProto()

	err = payload.EncodeWakuMessage(protoMsg, keyInfo)
	if err != nil {
		return err
	}

	topic := args.Topic
	if topic == "" {
		topic = relay.DefaultWakuTopic
	}

	_, err = p.node.Relay().PublishToTopic(req.Context(), protoMsg, topic)
	if err != nil {
		return err
	}

	*reply = true
	return nil
}

func (p *PrivateService) PostV1AsymmetricMessage(req *http.Request, args *AsymmetricMessageArgs, reply *bool) error {
	keyInfo := new(payload.KeyInfo)
	keyInfo.Kind = payload.Asymmetric

	pubKeyBytes, err := utils.DecodeHexString(args.PublicKey)
	if err != nil {
		return fmt.Errorf("public key cannot be decoded: %v", err)
	}

	pubKey, err := crypto.UnmarshalPubkey(pubKeyBytes)
	if err != nil {
		return fmt.Errorf("public key cannot be unmarshalled: %v", err)
	}

	keyInfo.PubKey = *pubKey

	msg := args.Message
	msg.Version = 1

	protoMsg := msg.toProto()

	err = payload.EncodeWakuMessage(protoMsg, keyInfo)
	if err != nil {
		return err
	}

	topic := args.Topic
	if topic == "" {
		topic = relay.DefaultWakuTopic
	}

	_, err = p.node.Relay().PublishToTopic(req.Context(), protoMsg, topic)
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

	symKeyBytes, err := utils.DecodeHexString(args.SymKey)
	if err != nil {
		return fmt.Errorf("invalid symmetric key: %w", err)
	}

	messages := make([]*pb.WakuMessage, len(p.messages[args.Topic]))
	copy(messages, p.messages[args.Topic])
	p.messages[args.Topic] = make([]*pb.WakuMessage, 0)

	var decodedMessages []*pb.WakuMessage
	for _, msg := range messages {
		err := payload.DecodeWakuMessage(msg, &payload.KeyInfo{
			Kind:   payload.Symmetric,
			SymKey: symKeyBytes,
		})
		if err != nil {
			continue
		}
		decodedMessages = append(decodedMessages, msg)
	}

	for i := range decodedMessages {
		*reply = append(*reply, ProtoToRPC(decodedMessages[i]))
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
		err := payload.DecodeWakuMessage(msg, &payload.KeyInfo{
			Kind:    payload.Asymmetric,
			PrivKey: privKey,
		})
		if err != nil {
			continue
		}
		decodedMessages = append(decodedMessages, msg)
	}

	for i := range decodedMessages {
		*reply = append(*reply, ProtoToRPC(decodedMessages[i]))
	}

	return nil
}

func (p *PrivateService) Start() {
	p.runner.Start()
}

func (p *PrivateService) Stop() {
	p.runner.Stop()
}
