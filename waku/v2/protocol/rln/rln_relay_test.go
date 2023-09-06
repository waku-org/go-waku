package rln

import (
	"context"
	"crypto/rand"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/suite"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln/group_manager"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln/group_manager/static"
	"github.com/waku-org/go-waku/waku/v2/timesource"
	"github.com/waku-org/go-waku/waku/v2/utils"
	r "github.com/waku-org/go-zerokit-rln/rln"
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
	s.Require().NoError(err)

	host, err := tests.MakeHost(context.Background(), port, rand.Reader)
	s.Require().NoError(err)

	relay := relay.NewWakuRelay(nil, 0, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, utils.Logger())
	relay.SetHost(host)
	err = relay.Start(context.Background())
	s.Require().NoError(err)
	defer relay.Stop()

	groupKeyPairs, root, err := r.CreateMembershipList(100)
	s.Require().NoError(err)

	var groupIDCommitments []r.IDCommitment
	for _, c := range groupKeyPairs {
		groupIDCommitments = append(groupIDCommitments, c.IDCommitment)
	}

	rlnInstance, rootTracker, err := GetRLNInstanceAndRootTracker("")
	s.Require().NoError(err)

	// index indicates the position of a membership key pair in the static list of group keys i.e., groupKeyPairs
	// the corresponding key pair will be used to mount rlnRelay on the current node
	// index also represents the index of the leaf in the Merkle tree that contains node's commitment key
	index := r.MembershipIndex(5)

	//
	idCredential := groupKeyPairs[index]
	groupManager, err := static.NewStaticGroupManager(groupIDCommitments, idCredential, index, rlnInstance, rootTracker, utils.Logger())
	s.Require().NoError(err)

	wakuRLNRelay := New(group_manager.GMDetails{
		GroupManager: groupManager,
		RootTracker:  rootTracker,
		RLN:          rlnInstance,
	}, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, utils.Logger())

	err = wakuRLNRelay.Start(context.TODO())
	s.Require().NoError(err)

	// get the root of Merkle tree which is constructed inside the mountRlnRelay proc
	calculatedRoot, err := wakuRLNRelay.RLN.GetMerkleRoot()
	s.Require().NoError(err)

	// Checks whether the Merkle tree is constructed correctly inside the mountRlnRelay func
	// this check is done by comparing the tree root resulted from mountRlnRelay i.e., calculatedRoot
	// against the root which is the expected root
	s.Require().Equal(root[:], calculatedRoot[:])
}

func (s *WakuRLNRelaySuite) TestUpdateLogAndHasDuplicate() {

	rlnInstance, err := r.NewRLN()
	s.Require().NoError(err)

	rootTracker, err := group_manager.NewMerkleRootTracker(acceptableRootWindowSize, rlnInstance)
	s.Require().NoError(err)

	rlnRelay := &WakuRLNRelay{
		nullifierLog: make(map[r.Nullifier][]r.ProofMetadata),
		GMDetails: group_manager.GMDetails{
			RootTracker: rootTracker,
		},
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

	msgProof1 := toRateLimitProof(wm1)
	msgProof2 := toRateLimitProof(wm2)
	msgProof3 := toRateLimitProof(wm3)

	md1, err := rlnInstance.ExtractMetadata(*msgProof1)
	s.Require().NoError(err)
	md2, err := rlnInstance.ExtractMetadata(*msgProof2)
	s.Require().NoError(err)
	md3, err := rlnInstance.ExtractMetadata(*msgProof3)
	s.Require().NoError(err)

	// check whether hasDuplicate correctly finds records with the same nullifiers but different secret shares
	// no duplicate for wm1 should be found, since the log is empty
	result1, err := rlnRelay.HasDuplicate(md1)
	s.Require().NoError(err)
	s.Require().False(result1) // No duplicate is found

	// Add it to the log
	added, err := rlnRelay.updateLog(md1)
	s.Require().NoError(err)
	s.Require().True(added)

	// no duplicate for wm2 should be found, its nullifier differs from wm1
	result2, err := rlnRelay.HasDuplicate(md2)
	s.Require().NoError(err)
	s.Require().False(result2) // No duplicate is found

	// Add it to the log
	added, err = rlnRelay.updateLog(md2)
	s.Require().NoError(err)
	s.Require().True(added)

	// wm3 has the same nullifier as wm1 but different secret shares, it should be detected as duplicate
	result3, err := rlnRelay.HasDuplicate(md3)
	s.Require().NoError(err)
	s.Require().True(result3) // It's a duplicate

}

func (s *WakuRLNRelaySuite) TestValidateMessage() {
	groupKeyPairs, _, err := r.CreateMembershipList(100)
	s.Require().NoError(err)

	var groupIDCommitments []r.IDCommitment
	for _, c := range groupKeyPairs {
		groupIDCommitments = append(groupIDCommitments, c.IDCommitment)
	}

	// index indicates the position of a membership key pair in the static list of group keys i.e., groupKeyPairs
	// the corresponding key pair will be used to mount rlnRelay on the current node
	// index also represents the index of the leaf in the Merkle tree that contains node's commitment key
	index := r.MembershipIndex(5)

	// Create a RLN instance
	rlnInstance, err := r.NewRLN()
	s.Require().NoError(err)
	//
	rootTracker, err := group_manager.NewMerkleRootTracker(acceptableRootWindowSize, rlnInstance)
	s.Require().NoError(err)
	//
	idCredential := groupKeyPairs[index]
	groupManager, err := static.NewStaticGroupManager(groupIDCommitments, idCredential, index, rlnInstance, rootTracker, utils.Logger())
	s.Require().NoError(err)

	rlnRelay := &WakuRLNRelay{
		GMDetails: group_manager.GMDetails{
			GroupManager: groupManager,
			RootTracker:  rootTracker,
			RLN:          rlnInstance,
		},

		nullifierLog: make(map[r.Nullifier][]r.ProofMetadata),
		log:          utils.Logger(),
		metrics:      newMetrics(prometheus.DefaultRegisterer),
	}

	//get the current epoch time
	now := time.Now()

	err = groupManager.Start(context.Background())
	s.Require().NoError(err)

	// create some messages from the same peer and append rln proof to them, except wm4

	wm1 := &pb.WakuMessage{Payload: []byte("Valid message")}
	err = rlnRelay.AppendRLNProof(wm1, now)
	s.Require().NoError(err)

	// another message in the same epoch as wm1, it will break the messaging rate limit
	wm2 := &pb.WakuMessage{Payload: []byte("Spam")}
	err = rlnRelay.AppendRLNProof(wm2, now)
	s.Require().NoError(err)

	// wm3 points to the next epoch
	wm3 := &pb.WakuMessage{Payload: []byte("Valid message")}
	err = rlnRelay.AppendRLNProof(wm3, now.Add(time.Second*time.Duration(r.EPOCH_UNIT_SECONDS)))
	s.Require().NoError(err)

	wm4 := &pb.WakuMessage{Payload: []byte("Invalid message")}

	// valid message
	msgValidate1, err := rlnRelay.ValidateMessage(wm1, &now)
	s.Require().NoError(err)

	// wm2 is published within the same Epoch as wm1 and should be found as spam
	msgValidate2, err := rlnRelay.ValidateMessage(wm2, &now)
	s.Require().NoError(err)

	// a valid message should be validated successfully
	msgValidate3, err := rlnRelay.ValidateMessage(wm3, &now)
	s.Require().NoError(err)

	// wm4 has no rln proof and should not be validated
	msgValidate4, err := rlnRelay.ValidateMessage(wm4, &now)
	s.Require().NoError(err)

	s.Require().Equal(validMessage, msgValidate1)
	s.Require().Equal(spamMessage, msgValidate2)
	s.Require().Equal(validMessage, msgValidate3)
	s.Require().Equal(invalidMessage, msgValidate4)
}
