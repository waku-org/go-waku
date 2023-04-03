package rln

import (
	"bytes"
	"context"
	"errors"
	"math"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/timesource"
	r "github.com/waku-org/go-zerokit-rln/rln"
	"go.uber.org/zap"
	proto "google.golang.org/protobuf/proto"
)

// the maximum clock difference between peers in seconds
const MAX_CLOCK_GAP_SECONDS = 20

// maximum allowed gap between the epochs of messages' RateLimitProofs
const MAX_EPOCH_GAP = int64(MAX_CLOCK_GAP_SECONDS / r.EPOCH_UNIT_SECONDS)

// Acceptable roots for merkle root validation of incoming messages
const AcceptableRootWindowSize = 5

type AppInfo struct {
	Application   string
	AppIdentifier string
	Version       string
}

type RegistrationHandler = func(tx *types.Transaction)

type SpamHandler = func(message *pb.WakuMessage) error

var RLNAppInfo = AppInfo{
	Application:   "go-waku-rln-relay",
	AppIdentifier: "01234567890abcdef",
	Version:       "0.1",
}

type GroupManager interface {
	Start(ctx context.Context, rln *r.RLN) error
	GenerateProof(input []byte, epoch r.Epoch) (*r.RateLimitProof, error)
	VerifyProof(input []byte, msgProof *r.RateLimitProof, ValidMerkleRoots ...r.MerkleNode) (bool, error)
	Stop()
}

type WakuRLNRelay struct {
	timesource timesource.Timesource

	groupManager GroupManager

	// pubsubTopic is the topic for which rln relay is mounted
	pubsubTopic  string
	contentTopic string
	relay        *relay.WakuRelay
	spamHandler  SpamHandler

	RLN *r.RLN

	validMerkleRoots []r.MerkleNode

	// the log of nullifiers and Shamir shares of the past messages grouped per epoch
	nullifierLogLock sync.RWMutex
	nullifierLog     map[r.Nullifier][]r.ProofMetadata

	log *zap.Logger
}

func New(
	relay *relay.WakuRelay,
	groupManager GroupManager,
	pubsubTopic string,
	contentTopic string,
	spamHandler SpamHandler,
	timesource timesource.Timesource,
	log *zap.Logger) (*WakuRLNRelay, error) {
	rlnInstance, err := r.NewRLN()
	if err != nil {
		return nil, err
	}

	// create the WakuRLNRelay
	rlnPeer := &WakuRLNRelay{
		RLN:          rlnInstance,
		groupManager: groupManager,
		pubsubTopic:  pubsubTopic,
		contentTopic: contentTopic,
		relay:        relay,
		spamHandler:  spamHandler,
		log:          log,
		timesource:   timesource,
		nullifierLog: make(map[r.MerkleNode][]r.ProofMetadata),
	}

	// TODO: pass RLN to group manager

	root, err := rlnPeer.RLN.GetMerkleRoot()
	if err != nil {
		return nil, err
	}

	rlnPeer.validMerkleRoots = append(rlnPeer.validMerkleRoots, root)

	return rlnPeer, nil
}

func (rln *WakuRLNRelay) Start(ctx context.Context) error {
	err := rln.groupManager.Start(ctx, rln.RLN)
	if err != nil {
		return err
	}

	root, err := rln.RLN.GetMerkleRoot()
	if err != nil {
		return err
	}

	rln.validMerkleRoots = append(rln.validMerkleRoots, root)

	// adds a topic validator for the supplied pubsub topic at the relay protocol
	// messages published on this pubsub topic will be relayed upon a successful validation, otherwise they will be dropped
	// the topic validator checks for the correct non-spamming proof of the message
	err = rln.addValidator(rln.relay, rln.pubsubTopic, rln.contentTopic, rln.spamHandler)
	if err != nil {
		return err
	}

	log.Info("rln relay topic validator mounted", zap.String("pubsubTopic", rln.pubsubTopic), zap.String("contentTopic", rln.contentTopic))

	return nil
}

func (rln *WakuRLNRelay) Stop() {
	rln.groupManager.Stop()
}

func (rln *WakuRLNRelay) HasDuplicate(proofMD r.ProofMetadata) (bool, error) {
	// returns true if there is another message in the  `nullifierLog` of the `rlnPeer` with the same
	// epoch and nullifier as `msg`'s epoch and nullifier but different Shamir secret shares
	// otherwise, returns false

	rln.nullifierLogLock.RLock()
	proofs, ok := rln.nullifierLog[proofMD.ExternalNullifier]
	rln.nullifierLogLock.RUnlock()

	// check if the epoch exists
	if !ok {
		return false, nil
	}

	for _, p := range proofs {
		if p.Equals(proofMD) {
			// there is an identical record, ignore rhe mag
			return false, nil
		}
	}

	// check for a message with the same nullifier but different secret shares
	matched := false
	for _, it := range proofs {
		if bytes.Equal(it.Nullifier[:], proofMD.Nullifier[:]) && (!bytes.Equal(it.ShareX[:], proofMD.ShareX[:]) || !bytes.Equal(it.ShareY[:], proofMD.ShareY[:])) {
			matched = true
			break
		}
	}

	return matched, nil
}

func (rln *WakuRLNRelay) updateLog(proofMD r.ProofMetadata) (bool, error) {
	rln.nullifierLogLock.Lock()
	defer rln.nullifierLogLock.Unlock()
	proofs, ok := rln.nullifierLog[proofMD.ExternalNullifier]

	// check if the epoch exists
	if !ok {
		rln.nullifierLog[proofMD.ExternalNullifier] = []r.ProofMetadata{proofMD}
		return true, nil
	}

	// check if an identical record exists
	for _, p := range proofs {
		if p.Equals(proofMD) {
			// TODO: slashing logic
			return true, nil
		}
	}

	// add proofMD to the log
	proofs = append(proofs, proofMD)
	rln.nullifierLog[proofMD.ExternalNullifier] = proofs

	return true, nil
}

func (rln *WakuRLNRelay) ValidateMessage(msg *pb.WakuMessage, optionalTime *time.Time) (MessageValidationResult, error) {
	// validate the supplied `msg` based on the waku-rln-relay routing protocol i.e.,
	// the `msg`'s epoch is within MAX_EPOCH_GAP of the current epoch
	// the `msg` has valid rate limit proof
	// the `msg` does not violate the rate limit
	// `timeOption` indicates Unix epoch time (fractional part holds sub-seconds)
	// if `timeOption` is supplied, then the current epoch is calculated based on that
	if msg == nil {
		return MessageValidationResult_Unknown, errors.New("nil message")
	}

	//  checks if the `msg`'s epoch is far from the current epoch
	// it corresponds to the validation of rln external nullifier
	var epoch r.Epoch
	if optionalTime != nil {
		epoch = r.CalcEpoch(*optionalTime)
	} else {
		// get current rln epoch
		epoch = r.CalcEpoch(rln.timesource.Now())
	}

	msgProof := ToRateLimitProof(msg)
	if msgProof == nil {
		// message does not contain a proof
		rln.log.Debug("invalid message: message does not contain a proof")
		return MessageValidationResult_Invalid, nil
	}

	proofMD, err := r.ExtractMetadata(*msgProof)
	if err != nil {
		rln.log.Debug("could not extract metadata", zap.Error(err))
		return MessageValidationResult_Invalid, nil
	}

	// calculate the gaps and validate the epoch
	gap := r.Diff(epoch, msgProof.Epoch)
	if int64(math.Abs(float64(gap))) > MAX_EPOCH_GAP {
		// message's epoch is too old or too ahead
		// accept messages whose epoch is within +-MAX_EPOCH_GAP from the current epoch
		rln.log.Debug("invalid message: epoch gap exceeds a threshold", zap.Int64("gap", gap))
		return MessageValidationResult_Invalid, nil
	}

	// verify the proof
	contentTopicBytes := []byte(msg.ContentTopic)
	input := append(msg.Payload, contentTopicBytes...)

	valid, err := rln.groupManager.VerifyProof(input, msgProof, rln.validMerkleRoots...)
	if err != nil {
		rln.log.Debug("could not verify proof", zap.Error(err))
		return MessageValidationResult_Invalid, nil
	}

	if !valid {
		// invalid proof
		rln.log.Debug("Invalid proof")
		return MessageValidationResult_Invalid, nil
	}

	// check if double messaging has happened
	hasDup, err := rln.HasDuplicate(proofMD)
	if err != nil {
		rln.log.Debug("validation error", zap.Error(err))
		return MessageValidationResult_Unknown, err
	}

	if hasDup {
		rln.log.Debug("spam received")
		return MessageValidationResult_Spam, nil
	}

	// insert the message to the log
	// the result of `updateLog` is discarded because message insertion is guaranteed by the implementation i.e.,
	// it will never error out
	_, err = rln.updateLog(proofMD)
	if err != nil {
		return MessageValidationResult_Unknown, err
	}

	rln.log.Debug("message is valid")
	return MessageValidationResult_Valid, nil
}

func (rln *WakuRLNRelay) AppendRLNProof(msg *pb.WakuMessage, senderEpochTime time.Time) error {
	// returns error if it could not create and append a `RateLimitProof` to the supplied `msg`
	// `senderEpochTime` indicates the number of seconds passed since Unix epoch. The fractional part holds sub-seconds.
	// The `epoch` field of `RateLimitProof` is derived from the provided `senderEpochTime` (using `calcEpoch()`)

	if msg == nil {
		return errors.New("nil message")
	}

	input := toRLNSignal(msg)

	proof, err := rln.groupManager.GenerateProof(input, r.CalcEpoch(senderEpochTime))
	if err != nil {
		return err
	}

	msg.RateLimitProof = &pb.RateLimitProof{
		Proof:         proof.Proof[:],
		MerkleRoot:    proof.MerkleRoot[:],
		Epoch:         proof.Epoch[:],
		ShareX:        proof.ShareX[:],
		ShareY:        proof.ShareY[:],
		Nullifier:     proof.Nullifier[:],
		RlnIdentifier: proof.RLNIdentifier[:],
	}

	return nil
}

func (r *WakuRLNRelay) insertMember(pubkey r.IDCommitment) error { // TODO: move to group manager? #########################################################
	r.log.Debug("a new key is added", zap.Binary("pubkey", pubkey[:]))
	// assuming all the members arrive in order
	err := r.RLN.InsertMember(pubkey)
	if err == nil {
		newRoot, err := r.RLN.GetMerkleRoot()
		if err != nil {
			r.log.Error("inserting member into merkletree", zap.Error(err))
			return err
		}
		r.validMerkleRoots = append(r.validMerkleRoots, newRoot)
		if len(r.validMerkleRoots) > AcceptableRootWindowSize {
			r.validMerkleRoots = r.validMerkleRoots[1:]
		}
	}

	return err
}

// this function sets a validator for the waku messages published on the supplied pubsubTopic and contentTopic
// if contentTopic is empty, then validation takes place for All the messages published on the given pubsubTopic
// the message validation logic is according to https://rfc.vac.dev/spec/17/
func (r *WakuRLNRelay) addValidator(
	relay *relay.WakuRelay,
	pubsubTopic string,
	contentTopic string,
	spamHandler SpamHandler) error {
	validator := func(ctx context.Context, peerID peer.ID, message *pubsub.Message) bool {
		r.log.Debug("rln-relay topic validator called")

		wakuMessage := &pb.WakuMessage{}
		if err := proto.Unmarshal(message.Data, wakuMessage); err != nil {
			r.log.Debug("could not unmarshal message")
			return true
		}

		// check the contentTopic
		if (wakuMessage.ContentTopic != "") && (contentTopic != "") && (wakuMessage.ContentTopic != contentTopic) {
			r.log.Debug("content topic did not match", zap.String("contentTopic", contentTopic))
			return true
		}

		// validate the message
		validationRes, err := r.ValidateMessage(wakuMessage, nil)
		if err != nil {
			r.log.Debug("validating message", zap.Error(err))
			return false
		}

		switch validationRes {
		case MessageValidationResult_Valid:
			r.log.Debug("message verified",
				zap.String("contentTopic", wakuMessage.ContentTopic),
				zap.Binary("epoch", wakuMessage.RateLimitProof.Epoch),
				zap.Int("timestamp", int(wakuMessage.Timestamp)),
				zap.Binary("payload", wakuMessage.Payload),
				zap.Any("proof", wakuMessage.RateLimitProof),
			)

			relay.AddToCache(pubsubTopic, message.ID, wakuMessage)

			return true
		case MessageValidationResult_Invalid:
			r.log.Debug("message could not be verified",
				zap.String("contentTopic", wakuMessage.ContentTopic),
				zap.Binary("epoch", wakuMessage.RateLimitProof.Epoch),
				zap.Int("timestamp", int(wakuMessage.Timestamp)),
				zap.Binary("payload", wakuMessage.Payload),
				zap.Any("proof", wakuMessage.RateLimitProof),
			)
			return false
		case MessageValidationResult_Spam:
			r.log.Debug("spam message found",
				zap.String("contentTopic", wakuMessage.ContentTopic),
				zap.Binary("epoch", wakuMessage.RateLimitProof.Epoch),
				zap.Int("timestamp", int(wakuMessage.Timestamp)),
				zap.Binary("payload", wakuMessage.Payload),
				zap.Any("proof", wakuMessage.RateLimitProof),
			)

			if spamHandler != nil {
				if err := spamHandler(wakuMessage); err != nil {
					r.log.Error("executing spam handler", zap.Error(err))
				}
			}

			return false
		default:
			r.log.Debug("unhandled validation result", zap.Int("validationResult", int(validationRes)))
			return false
		}
	}

	// In case there's a topic validator registered
	_ = relay.PubSub().UnregisterTopicValidator(pubsubTopic)

	return relay.PubSub().RegisterTopicValidator(pubsubTopic, validator)
}

func toRLNSignal(wakuMessage *pb.WakuMessage) []byte {
	if wakuMessage == nil {
		return []byte{}
	}

	contentTopicBytes := []byte(wakuMessage.ContentTopic)
	return append(wakuMessage.Payload, contentTopicBytes...)
}

func ToRateLimitProof(msg *pb.WakuMessage) *r.RateLimitProof {
	if msg == nil || msg.RateLimitProof == nil {
		return nil
	}

	result := &r.RateLimitProof{
		Proof:         r.ZKSNARK(r.Bytes128(msg.RateLimitProof.Proof)),
		MerkleRoot:    r.MerkleNode(r.Bytes32(msg.RateLimitProof.MerkleRoot)),
		Epoch:         r.Epoch(r.Bytes32(msg.RateLimitProof.Epoch)),
		ShareX:        r.MerkleNode(r.Bytes32(msg.RateLimitProof.ShareX)),
		ShareY:        r.MerkleNode(r.Bytes32(msg.RateLimitProof.ShareY)),
		Nullifier:     r.Nullifier(r.Bytes32(msg.RateLimitProof.Nullifier)),
		RLNIdentifier: r.RLNIdentifier(r.Bytes32(msg.RateLimitProof.RlnIdentifier)),
	}

	return result
}
