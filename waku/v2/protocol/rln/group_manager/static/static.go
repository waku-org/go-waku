package static

import (
	"context"
	"errors"

	"github.com/waku-org/go-zerokit-rln/rln"
	"go.uber.org/zap"
)

type StaticGroupManager struct {
	rln *rln.RLN
	log *zap.Logger

	group []rln.IDCommitment
}

func NewStaticGroupManager(
	group []rln.IDCommitment,
	memKeyPair rln.IdentityCredential,
	memIndex rln.MembershipIndex,
	log *zap.Logger,
) (*StaticGroupManager, error) {
	// check the peer's index and the inclusion of user's identity commitment in the group
	if memKeyPair.IDCommitment != group[int(memIndex)] {
		return nil, errors.New("peer's IDCommitment does not match commitment in group")
	}

	return &StaticGroupManager{
		log:   log.Named("rln-static"),
		group: group,
	}, nil
}

func (gm *StaticGroupManager) Start(ctx context.Context, rln *rln.RLN) error {
	gm.log.Info("mounting rln-relay in off-chain/static mode")

	gm.rln = rln

	// add members to the Merkle tree
	for _, member := range gm.group {
		if err := rln.InsertMember(member); err != nil {
			return err
		}
	}

	gm.group = nil // Deleting group to release memory

	return nil
}

func Setup(index rln.MembershipIndex) ([]rln.IDCommitment, rln.IdentityCredential, error) {
	// static group
	groupKeys := rln.STATIC_GROUP_KEYS
	groupSize := rln.STATIC_GROUP_SIZE

	// validate the user-supplied membership index
	if index >= rln.MembershipIndex(groupSize) {
		return nil, rln.IdentityCredential{}, errors.New("wrong membership index")
	}

	// create a sequence of MembershipKeyPairs from the group keys (group keys are in string format)
	credentials, err := rln.ToIdentityCredentials(groupKeys)
	if err != nil {
		return nil, rln.IdentityCredential{}, errors.New("invalid data on group keypairs")
	}

	// extract id commitment keys
	var groupOpt []rln.IDCommitment
	for _, c := range credentials {
		groupOpt = append(groupOpt, c.IDCommitment)
	}

	return groupOpt, credentials[index], nil
}

func (gm *StaticGroupManager) Stop() {
	// TODO:
}

func (gm *StaticGroupManager) GenerateProof(input []byte, epoch rln.Epoch) (*rln.RateLimitProof, error) {
	return nil, nil // TODO
}

func (gm *StaticGroupManager) VerifyProof(input []byte, msgProof *rln.RateLimitProof, ValidMerkleRoots ...rln.MerkleNode) (bool, error) {
	return false, nil
}
