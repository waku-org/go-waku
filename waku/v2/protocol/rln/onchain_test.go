//go:build include_onchain_tests
// +build include_onchain_tests

package rln

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"errors"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/waku-org/go-zerokit-rln/rln"
	r "github.com/waku-org/go-zerokit-rln/rln"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/suite"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln/contracts"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln/group_manager"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln/group_manager/dynamic"
	"github.com/waku-org/go-waku/waku/v2/timesource"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

var MEMBERSHIP_FEE = big.NewInt(1000000000000000) // wei - 0.001 eth

func TestWakuRLNRelayDynamicSuite(t *testing.T) {
	suite.Run(t, new(WakuRLNRelayDynamicSuite))
}

type WakuRLNRelayDynamicSuite struct {
	suite.Suite

	clientAddr string

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

	s.clientAddr = os.Getenv("GANACHE_NETWORK_RPC_URL")
	if s.clientAddr == "" {
		s.clientAddr = "ws://localhost:8545"
	}

	backend, err := ethclient.Dial(s.clientAddr)
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

func (s *WakuRLNRelayDynamicSuite) register(privKey *ecdsa.PrivateKey, commitment *big.Int, handler func(evt *contracts.RLNMemberRegistered) error) {
	auth, err := bind.NewKeyedTransactorWithChainID(privKey, s.chainID)
	s.Require().NoError(err)

	auth.Value = MEMBERSHIP_FEE
	auth.Context = context.TODO()

	tx, err := s.rlnContract.Register(auth, commitment)
	s.Require().NoError(err)
	receipt, err := bind.WaitMined(context.TODO(), s.backend, tx)
	s.Require().NoError(err)

	evt, err := s.rlnContract.ParseMemberRegistered(*receipt.Logs[0])
	s.Require().NoError(err)

	if handler != nil {
		err = handler(evt)
		s.Require().NoError(err)
	}
}

func (s *WakuRLNRelayDynamicSuite) TestDynamicGroupManagement() {
	// Create a RLN instance
	rlnInstance, err := r.NewRLN()
	s.Require().NoError(err)

	port, err := tests.FindFreePort(s.T(), "", 5)
	s.Require().NoError(err)

	host, err := tests.MakeHost(context.Background(), port, rand.Reader)
	s.Require().NoError(err)

	relay := relay.NewWakuRelay(nil, 0, timesource.NewDefaultClock(), utils.Logger())
	relay.SetHost(host)
	err = relay.Start(context.TODO())
	defer relay.Stop()
	s.Require().NoError(err)

	rt, err := group_manager.NewMerkleRootTracker(5, rlnInstance)
	s.Require().NoError(err)

	gm, err := dynamic.NewDynamicGroupManager(s.clientAddr, s.u1PrivKey, s.rlnAddr, "./test_onchain.json", "", false, nil, utils.Logger())
	s.Require().NoError(err)

	// initialize the WakuRLNRelay
	rlnRelay := &WakuRLNRelay{
		rootTracker:  rt,
		groupManager: gm,
		relay:        relay,
		RLN:          rlnInstance,
		log:          utils.Logger(),
		nullifierLog: make(map[r.MerkleNode][]r.ProofMetadata),
	}

	// generate another membership key pair
	keyPair2, err := rlnInstance.MembershipKeyGen()
	s.Require().NoError(err)

	err = rlnRelay.Start(context.Background())
	s.Require().NoError(err)

	// register user
	gm.Register(context.TODO())

	handler := func(evt *contracts.RLNMemberRegistered) error {
		pubkey := rln.Bytes32(evt.Pubkey.Bytes())
		if !bytes.Equal(pubkey[:], keyPair2.IDCommitment[:]) {
			return errors.New("not found")
		}

		return rlnInstance.InsertMember(pubkey)
	}

	// register member with contract
	s.register(s.u2PrivKey, dynamic.ToBigInt(keyPair2.IDCommitment[:]), handler)

	time.Sleep(2 * time.Second)
}

func (s *WakuRLNRelayDynamicSuite) TestInsertKeyMembershipContract() {

	s.register(s.u1PrivKey, big.NewInt(20), nil)

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

func (s *WakuRLNRelayDynamicSuite) TestMerkleTreeConstruction() {
	// Create a RLN instance
	rlnInstance, err := r.NewRLN()
	s.Require().NoError(err)

	keyPair1, err := rlnInstance.MembershipKeyGen()
	s.Require().NoError(err)

	keyPair2, err := rlnInstance.MembershipKeyGen()
	s.Require().NoError(err)

	err = rlnInstance.InsertMember(keyPair1.IDCommitment)
	s.Require().NoError(err)

	err = rlnInstance.InsertMember(keyPair2.IDCommitment)
	s.Require().NoError(err)

	//  get the Merkle root
	expectedRoot, err := rlnInstance.GetMerkleRoot()
	s.Require().NoError(err)

	// register the members to the contract
	s.register(s.u1PrivKey, dynamic.ToBigInt(keyPair1.IDCommitment[:]), nil)
	s.register(s.u1PrivKey, dynamic.ToBigInt(keyPair2.IDCommitment[:]), nil)

	// Creating relay
	port, err := tests.FindFreePort(s.T(), "", 5)
	s.Require().NoError(err)

	host, err := tests.MakeHost(context.Background(), port, rand.Reader)
	s.Require().NoError(err)

	relay := relay.NewWakuRelay(nil, 0, timesource.NewDefaultClock(), utils.Logger())
	relay.SetHost(host)
	err = relay.Start(context.TODO())
	defer relay.Stop()
	s.Require().NoError(err)

	// Subscribing to topic

	sub, err := relay.SubscribeToTopic(context.TODO(), RLNRELAY_PUBSUB_TOPIC)
	s.Require().NoError(err)
	defer sub.Unsubscribe()

	// mount the rln relay protocol in the on-chain/dynamic mode
	gm, err := dynamic.NewDynamicGroupManager(s.clientAddr, s.u1PrivKey, s.rlnAddr, "./test_onchain.json", "", false, nil, utils.Logger())
	s.Require().NoError(err)

	rlnRelay, err := New(relay, gm, RLNRELAY_PUBSUB_TOPIC, RLNRELAY_CONTENT_TOPIC, nil, timesource.NewDefaultClock(), utils.Logger())
	s.Require().NoError(err)

	// PreRegistering the keypair
	membershipIndex := rln.MembershipIndex(0)
	gm.SetCredentials(keyPair1, &membershipIndex)

	err = rlnRelay.Start(context.TODO())
	s.Require().NoError(err)

	// wait for the event to reach the group handler
	time.Sleep(2 * time.Second)

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

	host1, err := tests.MakeHost(context.Background(), port1, rand.Reader)
	s.Require().NoError(err)

	relay1 := relay.NewWakuRelay(nil, 0, timesource.NewDefaultClock(), utils.Logger())
	relay1.SetHost(host1)
	err = relay1.Start(context.TODO())
	defer relay1.Stop()
	s.Require().NoError(err)

	sub1, err := relay1.SubscribeToTopic(context.TODO(), RLNRELAY_PUBSUB_TOPIC)
	s.Require().NoError(err)
	defer sub1.Unsubscribe()

	// mount the rln relay protocol in the on-chain/dynamic mode
	gm1, err := dynamic.NewDynamicGroupManager(s.clientAddr, s.u1PrivKey, s.rlnAddr, "./test_onchain.json", "", false, nil, utils.Logger())
	s.Require().NoError(err)

	rlnRelay1, err := New(relay1, gm1, RLNRELAY_PUBSUB_TOPIC, RLNRELAY_CONTENT_TOPIC, nil, timesource.NewDefaultClock(), utils.Logger())
	s.Require().NoError(err)
	err = rlnRelay1.Start(context.TODO())
	s.Require().NoError(err)

	// Node 2 ============================================================
	port2, err := tests.FindFreePort(s.T(), "", 5)
	s.Require().NoError(err)

	host2, err := tests.MakeHost(context.Background(), port2, rand.Reader)
	s.Require().NoError(err)

	relay2 := relay.NewWakuRelay(nil, 0, timesource.NewDefaultClock(), utils.Logger())
	relay2.SetHost(host2)
	err = relay2.Start(context.TODO())
	defer relay2.Stop()
	s.Require().NoError(err)

	sub2, err := relay2.SubscribeToTopic(context.TODO(), RLNRELAY_PUBSUB_TOPIC)
	s.Require().NoError(err)
	defer sub2.Unsubscribe()

	// mount the rln relay protocol in the on-chain/dynamic mode
	gm2, err := dynamic.NewDynamicGroupManager(s.clientAddr, s.u2PrivKey, s.rlnAddr, "./test_onchain.json", "", false, nil, utils.Logger())
	s.Require().NoError(err)

	rlnRelay2, err := New(relay2, gm2, RLNRELAY_PUBSUB_TOPIC, RLNRELAY_CONTENT_TOPIC, nil, timesource.NewDefaultClock(), utils.Logger())
	s.Require().NoError(err)
	err = rlnRelay2.Start(context.TODO())
	s.Require().NoError(err)

	// ==================================
	// the two nodes should be registered into the contract
	// since nodes are spun up sequentially
	// the first node has index 0 whereas the second node gets index 1
	idx1, err := rlnRelay1.groupManager.MembershipIndex()
	s.Require().NoError(err)
	idx2, err := rlnRelay2.groupManager.MembershipIndex()
	s.Require().NoError(err)

	s.Require().Equal(r.MembershipIndex(0), idx1)
	s.Require().Equal(r.MembershipIndex(1), idx2)
}
