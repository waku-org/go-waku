package rln

import (
	"context"
	"crypto/rand"
	"testing"
	"time"

	r "github.com/decanus/go-rln/rln"
	"github.com/status-im/go-waku/tests"
	"github.com/status-im/go-waku/waku/v2/protocol/pb"
	"github.com/status-im/go-waku/waku/v2/protocol/relay"
	"github.com/status-im/go-waku/waku/v2/utils"
	"github.com/stretchr/testify/suite"
)

const RLNRELAY_PUBSUB_TOPIC = "waku/2/rlnrelay/proto"
const RLNRELAY_CONTENT_TOPIC = "waku/2/rlnrelay/proto"

func TestWakuRLNRelaySuite(t *testing.T) {
	suite.Run(t, new(WakuRLNRelaySuite))
}

type WakuRLNRelaySuite struct {
	suite.Suite
}

func (s *WakuRLNRelaySuite) TestOffchainMode() {
	port, err := tests.FindFreePort(s.T(), "", 5)
	s.NoError(err)

	host, err := tests.MakeHost(context.Background(), port, rand.Reader)
	s.NoError(err)

	relay, err := relay.NewWakuRelay(context.Background(), host, nil, 0, utils.Logger())
	defer relay.Stop()
	s.NoError(err)

	params, err := parametersKeyBytes()
	s.NoError(err)

	groupKeyPairs, root, err := r.CreateMembershipList(100, params)
	s.NoError(err)

	var groupIDCommitments []r.IDCommitment
	for _, c := range groupKeyPairs {
		groupIDCommitments = append(groupIDCommitments, c.IDCommitment)
	}

	// index indicates the position of a membership key pair in the static list of group keys i.e., groupKeyPairs
	// the corresponding key pair will be used to mount rlnRelay on the current node
	// index also represents the index of the leaf in the Merkle tree that contains node's commitment key
	index := r.MembershipIndex(5)

	wakuRLNRelay, err := RlnRelayStatic(relay, groupIDCommitments, groupKeyPairs[index], index, RLNRELAY_PUBSUB_TOPIC, RLNRELAY_CONTENT_TOPIC, nil, utils.Logger())
	s.NoError(err)

	// get the root of Merkle tree which is constructed inside the mountRlnRelay proc
	calculatedRoot, err := wakuRLNRelay.RLN.GetMerkleRoot()
	s.NoError(err)

	// Checks whether the Merkle tree is constructed correctly inside the mountRlnRelay func
	// this check is done by comparing the tree root resulted from mountRlnRelay i.e., calculatedRoot
	// against the root which is the expected root
	s.Equal(root[:], calculatedRoot[:])
}

func (s *WakuRLNRelaySuite) TestUpdateLogAndHasDuplicate() {

	rlnRelay := &WakuRLNRelay{
		nullifierLog: make(map[r.Epoch][]r.ProofMetadata),
	}

	epoch := r.GetCurrentEpoch()

	//  create some dummy nullifiers and secret shares
	var nullifier1, nullifier2, nullifier3 r.Nullifier
	var shareX1, shareX2, shareX3 r.MerkleNode
	var shareY1, shareY2, shareY3 r.MerkleNode
	for i := 0; i < 32; i++ {
		nullifier1[i] = 1
		nullifier2[i] = 2
		nullifier3[i] = nullifier1[i]
		shareX1[i] = 1
		shareX2[i] = 2
		shareX3[i] = 3
		shareY1[i] = 1
		shareY2[i] = shareX2[i]
		shareY3[i] = shareX3[i]
	}

	wm1 := &pb.WakuMessage{RateLimitProof: &pb.RateLimitProof{Epoch: epoch[:], Nullifier: nullifier1[:], ShareX: shareX1[:], ShareY: shareY1[:]}}
	wm2 := &pb.WakuMessage{RateLimitProof: &pb.RateLimitProof{Epoch: epoch[:], Nullifier: nullifier2[:], ShareX: shareX2[:], ShareY: shareY2[:]}}
	wm3 := &pb.WakuMessage{RateLimitProof: &pb.RateLimitProof{Epoch: epoch[:], Nullifier: nullifier3[:], ShareX: shareX3[:], ShareY: shareY3[:]}}

	// check whether hasDuplicate correctly finds records with the same nullifiers but different secret shares
	// no duplicate for wm1 should be found, since the log is empty
	result1, err := rlnRelay.HasDuplicate(wm1)
	s.NoError(err)
	s.False(result1) // No duplicate is found

	// Add it to the log
	added, err := rlnRelay.UpdateLog(wm1)
	s.NoError(err)
	s.True(added)

	// no duplicate for wm2 should be found, its nullifier differs from wm1
	result2, err := rlnRelay.HasDuplicate(wm2)
	s.NoError(err)
	s.False(result2) // No duplicate is found

	// Add it to the log
	added, err = rlnRelay.UpdateLog(wm2)
	s.NoError(err)
	s.True(added)

	// wm3 has the same nullifier as wm1 but different secret shares, it should be detected as duplicate
	result3, err := rlnRelay.HasDuplicate(wm3)
	s.NoError(err)
	s.True(result3) // It's a duplicate

}

func (s *WakuRLNRelaySuite) TestValidateMessage() {
	params, err := parametersKeyBytes()
	s.NoError(err)

	groupKeyPairs, _, err := r.CreateMembershipList(100, params)
	s.NoError(err)

	var groupIDCommitments []r.IDCommitment
	for _, c := range groupKeyPairs {
		groupIDCommitments = append(groupIDCommitments, c.IDCommitment)
	}

	// index indicates the position of a membership key pair in the static list of group keys i.e., groupKeyPairs
	// the corresponding key pair will be used to mount rlnRelay on the current node
	// index also represents the index of the leaf in the Merkle tree that contains node's commitment key
	index := r.MembershipIndex(5)

	// Create a RLN instance
	rlnInstance, err := r.NewRLN(params)
	s.NoError(err)

	added := rlnInstance.AddAll(groupIDCommitments)
	s.True(added)

	rlnRelay := &WakuRLNRelay{
		membershipIndex:   index,
		membershipKeyPair: groupKeyPairs[index],
		RLN:               rlnInstance,
		nullifierLog:      make(map[r.Epoch][]r.ProofMetadata),
		log:               utils.Logger(),
	}

	//get the current epoch time
	now := time.Now()

	// create some messages from the same peer and append rln proof to them, except wm4

	wm1 := &pb.WakuMessage{Payload: []byte("Valid message")}
	err = rlnRelay.AppendRLNProof(wm1, now)
	s.NoError(err)

	// another message in the same epoch as wm1, it will break the messaging rate limit
	wm2 := &pb.WakuMessage{Payload: []byte("Spam")}
	err = rlnRelay.AppendRLNProof(wm2, now)
	s.NoError(err)

	// wm3 points to the next epoch
	wm3 := &pb.WakuMessage{Payload: []byte("Valid message")}
	err = rlnRelay.AppendRLNProof(wm3, now.Add(time.Second*time.Duration(r.EPOCH_UNIT_SECONDS)))
	s.NoError(err)

	wm4 := &pb.WakuMessage{Payload: []byte("Invalid message")}

	// valid message
	msgValidate1, err := rlnRelay.ValidateMessage(wm1, &now)
	s.NoError(err)

	// wm2 is published within the same Epoch as wm1 and should be found as spam
	msgValidate2, err := rlnRelay.ValidateMessage(wm2, &now)
	s.NoError(err)

	// a valid message should be validated successfully
	msgValidate3, err := rlnRelay.ValidateMessage(wm3, &now)
	s.NoError(err)

	// wm4 has no rln proof and should not be validated
	msgValidate4, err := rlnRelay.ValidateMessage(wm4, &now)
	s.NoError(err)

	s.Equal(MessageValidationResult_Valid, msgValidate1)
	s.Equal(MessageValidationResult_Spam, msgValidate2)
	s.Equal(MessageValidationResult_Valid, msgValidate3)
	s.Equal(MessageValidationResult_Invalid, msgValidate4)
}
