package rln

import (
	"errors"

	r "github.com/decanus/go-rln/rln"
	"github.com/status-im/go-waku/waku/v2/protocol/relay"
	"go.uber.org/zap"
)

func RlnRelayStatic(
	relay *relay.WakuRelay,
	group []r.IDCommitment,
	memKeyPair r.MembershipKeyPair,
	memIndex r.MembershipIndex,
	pubsubTopic string,
	contentTopic string,
	spamHandler SpamHandler,
	log *zap.Logger,
) (*WakuRLNRelay, error) {
	log = log.Named("rln-static")

	log.Info("mounting rln-relay in off-chain/static mode")

	// check the peer's index and the inclusion of user's identity commitment in the group
	if memKeyPair.IDCommitment != group[int(memIndex)] {
		return nil, errors.New("peer's IDCommitment does not match commitment in group")
	}

	// create an RLN instance
	parameters, err := parametersKeyBytes()
	if err != nil {
		return nil, err
	}

	rlnInstance, err := r.NewRLN(parameters)
	if err != nil {
		return nil, err
	}

	// add members to the Merkle tree
	for _, member := range group {
		if !rlnInstance.InsertMember(member) {
			return nil, errors.New("could not add member")
		}
	}

	// create the WakuRLNRelay
	rlnPeer := &WakuRLNRelay{
		membershipKeyPair: memKeyPair,
		membershipIndex:   memIndex,
		RLN:               rlnInstance,
		pubsubTopic:       pubsubTopic,
		contentTopic:      contentTopic,
		log:               log,
		nullifierLog:      make(map[r.Epoch][]r.ProofMetadata),
	}

	// adds a topic validator for the supplied pubsub topic at the relay protocol
	// messages published on this pubsub topic will be relayed upon a successful validation, otherwise they will be dropped
	// the topic validator checks for the correct non-spamming proof of the message
	err = rlnPeer.addValidator(relay, pubsubTopic, contentTopic, spamHandler)
	if err != nil {
		return nil, err
	}

	log.Info("rln relay topic validator mounted", zap.String("pubsubTopic", pubsubTopic), zap.String("contentTopic", contentTopic))

	return rlnPeer, nil
}
