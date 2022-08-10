package rln

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"errors"
	"math/big"
	"sync"
	"testing"
	"time"

	r "github.com/status-im/go-rln/rln"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/status-im/go-waku/tests"
	"github.com/status-im/go-waku/waku/v2/protocol/relay"
	"github.com/status-im/go-waku/waku/v2/protocol/rln/contracts"
	"github.com/status-im/go-waku/waku/v2/utils"
	"github.com/stretchr/testify/suite"
)

const ETH_CLIENT_ADDRESS = "ws://localhost:8545"

func TestWakuRLNRelayDynamicSuite(t *testing.T) {
	suite.Run(t, new(WakuRLNRelayDynamicSuite))
}

type WakuRLNRelayDynamicSuite struct {
	suite.Suite

	backend     *ethclient.Client
	chainID     *big.Int
	rlnAddr     common.Address
	rlnContract *contracts.RLN

	u1PrivKey *ecdsa.PrivateKey
	u2PrivKey *ecdsa.PrivateKey
	u3PrivKey *ecdsa.PrivateKey
	u4PrivKey *ecdsa.PrivateKey
	u5PrivKey *ecdsa.PrivateKey
}

func (s *WakuRLNRelayDynamicSuite) SetupTest() {
	backend, err := ethclient.Dial(ETH_CLIENT_ADDRESS)
	s.Require().NoError(err)

	chainID, err := backend.ChainID(context.TODO())
	s.Require().NoError(err)

	// TODO: obtain account list from ganache mnemonic or from eth_accounts

	s.u1PrivKey, err = crypto.ToECDSA(common.FromHex("0x156ec84a451d8a2d0062993242b6c4e863647f5544ff8030f23578d4142f43f8"))
	s.Require().NoError(err)
	s.u2PrivKey, err = crypto.ToECDSA(common.FromHex("0xa00da43843ad6b5161ddbace48f293ac3f82f8a8257af34de4c32900bb6e9a97"))
	s.Require().NoError(err)
	s.u3PrivKey, err = crypto.ToECDSA(common.FromHex("0xa4c8d3ed78cd722521fac9d734c45187a4f5e887570be1f707a7bbce054c01ea"))
	s.Require().NoError(err)
	s.u4PrivKey, err = crypto.ToECDSA(common.FromHex("0x6b11ba548a7fd1958eb156877cc7bdd02d99d876b55381aa9b106c16b0b7a805"))
	s.Require().NoError(err)
	s.u5PrivKey, err = crypto.ToECDSA(common.FromHex("0x0410196287d0af405e5c16f610de52416bd48be74836dbca93d73e24bffb5a81"))
	s.Require().NoError(err)

	s.backend = backend
	s.chainID = chainID

	// Deploying contracts
	auth, err := bind.NewKeyedTransactorWithChainID(s.u1PrivKey, chainID)
	s.Require().NoError(err)

	poseidonHasherAddr, _, _, err := contracts.DeployPoseidonHasher(auth, backend)
	s.Require().NoError(err)

	rlnAddr, _, rlnContract, err := contracts.DeployRLN(auth, backend, MEMBERSHIP_FEE, big.NewInt(20), poseidonHasherAddr)
	s.Require().NoError(err)

	s.rlnAddr = rlnAddr
	s.rlnContract = rlnContract
}

func (s *WakuRLNRelayDynamicSuite) register(privKey *ecdsa.PrivateKey, commitment *big.Int) {
	auth, err := bind.NewKeyedTransactorWithChainID(privKey, s.chainID)
	s.Require().NoError(err)

	auth.Value = MEMBERSHIP_FEE
	auth.Context = context.TODO()

	tx, err := s.rlnContract.Register(auth, commitment)
	s.Require().NoError(err)
	_, err = bind.WaitMined(context.TODO(), s.backend, tx)
	s.Require().NoError(err)
}

func (s *WakuRLNRelayDynamicSuite) TestDynamicGroupManagement() {
	params, err := parametersKeyBytes()
	s.Require().NoError(err)

	// Create a RLN instance
	rlnInstance, err := r.NewRLN(params)
	s.Require().NoError(err)

	keyPair, err := rlnInstance.MembershipKeyGen()
	s.Require().NoError(err)

	// initialize the WakuRLNRelay
	rlnPeer := &WakuRLNRelay{
		ctx:                       context.TODO(),
		membershipIndex:           r.MembershipIndex(0),
		membershipContractAddress: s.rlnAddr,
		ethClientAddress:          ETH_CLIENT_ADDRESS,
		ethAccountPrivateKey:      s.u1PrivKey,
		RLN:                       rlnInstance,
		log:                       utils.Logger(),
		nullifierLog:              make(map[r.Epoch][]r.ProofMetadata),
		membershipKeyPair:         *keyPair,
	}

	// generate another membership key pair
	keyPair2, err := rlnInstance.MembershipKeyGen()
	s.Require().NoError(err)

	wg := &sync.WaitGroup{}
	wg.Add(2)

	handler := func(pubkey r.IDCommitment, index r.MembershipIndex) error {
		if pubkey == keyPair.IDCommitment || pubkey == keyPair2.IDCommitment {
			wg.Done()
		}

		if !rlnInstance.InsertMember(pubkey) {
			return errors.New("couldn't insert member")
		}

		return nil
	}

	// mount the handler for listening to the contract events
	go rlnPeer.HandleGroupUpdates(handler)

	// Register first member
	s.register(s.u1PrivKey, toBigInt(keyPair.IDCommitment[:]))

	// Register second member
	s.register(s.u2PrivKey, toBigInt(keyPair2.IDCommitment[:]))

	wg.Wait()
}

func (s *WakuRLNRelayDynamicSuite) TestInsertKeyMembershipContract() {

	s.register(s.u1PrivKey, big.NewInt(20))

	// Batch Register
	auth, err := bind.NewKeyedTransactorWithChainID(s.u2PrivKey, s.chainID)
	s.Require().NoError(err)

	auth.Value = MEMBERSHIP_FEE.Mul(big.NewInt(2), MEMBERSHIP_FEE)
	auth.Context = context.TODO()

	tx, err := s.rlnContract.RegisterBatch(auth, []*big.Int{big.NewInt(20), big.NewInt(21)})
	s.Require().NoError(err)

	_, err = bind.WaitMined(context.TODO(), s.backend, tx)
	s.Require().NoError(err)
}

func (s *WakuRLNRelayDynamicSuite) TestRegistrationProcedure() {
	params, err := parametersKeyBytes()
	s.Require().NoError(err)

	// Create a RLN instance
	rlnInstance, err := r.NewRLN(params)
	s.Require().NoError(err)

	keyPair, err := rlnInstance.MembershipKeyGen()
	s.Require().NoError(err)

	// initialize the WakuRLNRelay
	rlnPeer := &WakuRLNRelay{
		ctx:                       context.TODO(),
		membershipIndex:           r.MembershipIndex(0),
		membershipContractAddress: s.rlnAddr,
		ethClientAddress:          ETH_CLIENT_ADDRESS,
		ethAccountPrivateKey:      s.u1PrivKey,
		RLN:                       rlnInstance,
		log:                       utils.Logger(),
		nullifierLog:              make(map[r.Epoch][]r.ProofMetadata),
		membershipKeyPair:         *keyPair,
	}

	_, err = rlnPeer.Register(context.TODO())
	s.Require().NoError(err)
}

func (s *WakuRLNRelayDynamicSuite) TestMerkleTreeConstruction() {
	params, err := parametersKeyBytes()
	s.Require().NoError(err)

	// Create a RLN instance
	rlnInstance, err := r.NewRLN(params)
	s.Require().NoError(err)

	keyPair1, err := rlnInstance.MembershipKeyGen()
	s.Require().NoError(err)

	keyPair2, err := rlnInstance.MembershipKeyGen()
	s.Require().NoError(err)

	r1 := rlnInstance.InsertMember(keyPair1.IDCommitment)
	r2 := rlnInstance.InsertMember(keyPair2.IDCommitment)
	s.Require().True(r1)
	s.Require().True(r2)

	//  get the Merkle root
	expectedRoot, err := rlnInstance.GetMerkleRoot()
	s.Require().NoError(err)

	// register the members to the contract
	s.register(s.u1PrivKey, toBigInt(keyPair1.IDCommitment[:]))
	s.register(s.u1PrivKey, toBigInt(keyPair2.IDCommitment[:]))

	port, err := tests.FindFreePort(s.T(), "", 5)
	s.Require().NoError(err)

	host, err := tests.MakeHost(context.TODO(), port, rand.Reader)
	s.Require().NoError(err)

	relay, err := relay.NewWakuRelay(context.TODO(), host, nil, 0, utils.Logger())
	defer relay.Stop()
	s.Require().NoError(err)

	sub, err := relay.SubscribeToTopic(context.TODO(), RLNRELAY_PUBSUB_TOPIC)
	s.Require().NoError(err)
	defer sub.Unsubscribe()

	// mount the rln relay protocol in the on-chain/dynamic mode
	rlnRelay, err := RlnRelayDynamic(context.TODO(), relay, ETH_CLIENT_ADDRESS, nil, s.rlnAddr, keyPair1, r.MembershipIndex(0), RLNRELAY_PUBSUB_TOPIC, RLNRELAY_CONTENT_TOPIC, nil, utils.Logger())
	s.Require().NoError(err)

	// wait for the event to reach the group handler
	time.Sleep(1 * time.Second)

	// rln pks are inserted into the rln peer's Merkle tree and the resulting root
	// is expected to be the same as the calculatedRoot i.e., the one calculated outside of the mountRlnRelayDynamic proc
	calculatedRoot, err := rlnRelay.RLN.GetMerkleRoot()
	s.Require().NoError(err)

	s.Require().Equal(expectedRoot, calculatedRoot)
}

func (s *WakuRLNRelayDynamicSuite) TestCorrectRegistrationOfPeers() {

	// Node 1 ============================================================
	port1, err := tests.FindFreePort(s.T(), "", 5)
	s.Require().NoError(err)

	host1, err := tests.MakeHost(context.TODO(), port1, rand.Reader)
	s.Require().NoError(err)

	relay1, err := relay.NewWakuRelay(context.TODO(), host1, nil, 0, utils.Logger())
	defer relay1.Stop()
	s.Require().NoError(err)

	sub1, err := relay1.SubscribeToTopic(context.TODO(), RLNRELAY_PUBSUB_TOPIC)
	s.Require().NoError(err)
	defer sub1.Unsubscribe()

	// mount the rln relay protocol in the on-chain/dynamic mode
	rlnRelay1, err := RlnRelayDynamic(context.TODO(), relay1, ETH_CLIENT_ADDRESS, s.u1PrivKey, s.rlnAddr, nil, r.MembershipIndex(0), RLNRELAY_PUBSUB_TOPIC, RLNRELAY_CONTENT_TOPIC, nil, utils.Logger())
	s.Require().NoError(err)

	// Node 2 ============================================================
	port2, err := tests.FindFreePort(s.T(), "", 5)
	s.Require().NoError(err)

	host2, err := tests.MakeHost(context.TODO(), port2, rand.Reader)
	s.Require().NoError(err)

	relay2, err := relay.NewWakuRelay(context.TODO(), host2, nil, 0, utils.Logger())
	defer relay2.Stop()
	s.Require().NoError(err)

	sub2, err := relay2.SubscribeToTopic(context.TODO(), RLNRELAY_PUBSUB_TOPIC)
	s.Require().NoError(err)
	defer sub2.Unsubscribe()

	// mount the rln relay protocol in the on-chain/dynamic mode
	rlnRelay2, err := RlnRelayDynamic(context.TODO(), relay2, ETH_CLIENT_ADDRESS, s.u2PrivKey, s.rlnAddr, nil, r.MembershipIndex(0), RLNRELAY_PUBSUB_TOPIC, RLNRELAY_CONTENT_TOPIC, nil, utils.Logger())
	s.Require().NoError(err)

	// ==================================
	// the two nodes should be registered into the contract
	// since nodes are spun up sequentially
	// the first node has index 0 whereas the second node gets index 1
	s.Require().Equal(r.MembershipIndex(0), rlnRelay1.membershipIndex)
	s.Require().Equal(r.MembershipIndex(1), rlnRelay2.membershipIndex)
}
