//go:build include_onchain_tests
// +build include_onchain_tests

package rln

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/waku-org/go-zerokit-rln/rln"
	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/suite"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln/contracts"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln/group_manager"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln/group_manager/dynamic"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln/keystore"
	"github.com/waku-org/go-waku/waku/v2/timesource"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

var membershipFee = big.NewInt(1000000000000000) // wei - 0.001 eth
const keystorePassword = "test"

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
}

// TODO: on teardown, remove credentials

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

	s.backend = backend
	s.chainID = chainID

	// Deploying contracts
	auth, err := bind.NewKeyedTransactorWithChainID(s.u1PrivKey, chainID)
	s.Require().NoError(err)

	// TODO: update rln contract

	poseidonHasherAddr, _, _, err := contracts.DeployPoseidonHasher(auth, backend)
	s.Require().NoError(err)

	rlnAddr, _, rlnContract, err := contracts.DeployRLN(auth, backend, membershipFee, big.NewInt(20), poseidonHasherAddr)
	s.Require().NoError(err)

	s.rlnAddr = rlnAddr
	s.rlnContract = rlnContract
}

func (s *WakuRLNRelayDynamicSuite) removeCredentials(path string) {
	err := os.Remove(path)
	if err != nil {
		utils.Logger().Warn("could not remove credentials", zap.String("path", path))
	}
}

func (s *WakuRLNRelayDynamicSuite) generateCredentials(rlnInstance *rln.RLN) *rln.IdentityCredential {
	identityCredential, err := rlnInstance.MembershipKeyGen()
	s.Require().NoError(err)
	return identityCredential
}

func (s *WakuRLNRelayDynamicSuite) register(identityCredential *rln.IdentityCredential, privKey *ecdsa.PrivateKey, keystorePath string) (rln.MembershipIndex, uint) {

	auth, err := bind.NewKeyedTransactorWithChainID(privKey, s.chainID)
	s.Require().NoError(err)

	auth.Value = membershipFee
	auth.Context = context.TODO()

	tx, err := s.rlnContract.Register(auth, rln.Bytes32ToBigInt(identityCredential.IDCommitment))
	s.Require().NoError(err)
	txReceipt, err := bind.WaitMined(context.TODO(), s.backend, tx)
	s.Require().NoError(err)

	s.Require().Equal(txReceipt.Status, types.ReceiptStatusSuccessful)

	evt, err := s.rlnContract.ParseMemberRegistered(*txReceipt.Logs[0])
	s.Require().NoError(err)

	membershipIndex := rln.MembershipIndex(uint(evt.Index.Int64()))

	membershipGroup := keystore.MembershipGroup{
		TreeIndex: membershipIndex,
		MembershipContract: keystore.MembershipContract{
			ChainId: fmt.Sprintf("0x%X", s.chainID.Int64()),
			Address: s.rlnAddr.String(),
		},
	}

	membershipGroupIndex, err := keystore.AddMembershipCredentials(keystorePath, identityCredential, membershipGroup, keystorePassword, dynamic.RLNAppInfo, keystore.DefaultSeparator)
	s.Require().NoError(err)

	return membershipIndex, membershipGroupIndex
}

func (s *WakuRLNRelayDynamicSuite) TestDynamicGroupManagement() {
	// Create a RLN instance
	rlnInstance, err := rln.NewRLN()
	s.Require().NoError(err)

	port, err := tests.FindFreePort(s.T(), "", 5)
	s.Require().NoError(err)

	host, err := tests.MakeHost(context.Background(), port, rand.Reader)
	s.Require().NoError(err)

	relay := relay.NewWakuRelay(nil, 0, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, utils.Logger())
	relay.SetHost(host)
	err = relay.Start(context.TODO())
	defer relay.Stop()
	s.Require().NoError(err)

	rt, err := group_manager.NewMerkleRootTracker(5, rlnInstance)
	s.Require().NoError(err)

	u1Credentials := s.generateCredentials(rlnInstance)
	keystorePath1 := "./test_onchain.json"
	_, membershipGroupIndex := s.register(u1Credentials, s.u1PrivKey, keystorePath1)
	defer s.removeCredentials(keystorePath1)

	gm, err := dynamic.NewDynamicGroupManager(s.clientAddr, s.rlnAddr, membershipGroupIndex, keystorePath1, keystorePassword, 0, false, utils.Logger())
	s.Require().NoError(err)

	// initialize the WakuRLNRelay
	rlnRelay := &WakuRLNRelay{
		rootTracker:  rt,
		groupManager: gm,
		relay:        relay,
		RLN:          rlnInstance,
		log:          utils.Logger(),
		nullifierLog: make(map[rln.MerkleNode][]rln.ProofMetadata),
	}

	err = rlnRelay.Start(context.TODO())
	s.Require().NoError(err)

	u2Credentials := s.generateCredentials(rlnInstance)
	keystorePath2 := "./test_onchain2.json"
	membershipIndex, _ := s.register(u2Credentials, s.u2PrivKey, keystorePath2)
	defer s.removeCredentials(keystorePath2)

	time.Sleep(1 * time.Second)

	treeCommitment, err := rlnInstance.GetLeaf(membershipIndex)
	s.Require().NoError(err)
	s.Require().Equal(u2Credentials.IDCommitment, treeCommitment)
}

func (s *WakuRLNRelayDynamicSuite) TestInsertKeyMembershipContract() {
	// Create a RLN instance
	rlnInstance, err := rln.NewRLN()
	s.Require().NoError(err)

	credentials1 := s.generateCredentials(rlnInstance)
	credentials2 := s.generateCredentials(rlnInstance)
	credentials3 := s.generateCredentials(rlnInstance)

	keystorePath1 := "./test_onchain.json"
	s.register(credentials1, s.u1PrivKey, keystorePath1)
	defer s.removeCredentials(keystorePath1)

	// Batch Register
	auth, err := bind.NewKeyedTransactorWithChainID(s.u2PrivKey, s.chainID)
	s.Require().NoError(err)

	auth.Value = membershipFee.Mul(big.NewInt(2), membershipFee)
	auth.Context = context.TODO()

	tx, err := s.rlnContract.RegisterBatch(auth, []*big.Int{rln.Bytes32ToBigInt(credentials2.IDCommitment), rln.Bytes32ToBigInt(credentials3.IDCommitment)})
	s.Require().NoError(err)

	txReceipt, err := bind.WaitMined(context.TODO(), s.backend, tx)
	s.Require().NoError(err)
	s.Require().Equal(txReceipt.Status, types.ReceiptStatusSuccessful)
}

func (s *WakuRLNRelayDynamicSuite) TestMerkleTreeConstruction() {
	// Create a RLN instance
	rlnInstance, err := rln.NewRLN()
	s.Require().NoError(err)

	credentials1 := s.generateCredentials(rlnInstance)
	credentials2 := s.generateCredentials(rlnInstance)

	err = rlnInstance.InsertMember(credentials1.IDCommitment)
	s.Require().NoError(err)

	err = rlnInstance.InsertMember(credentials2.IDCommitment)
	s.Require().NoError(err)

	//  get the Merkle root
	expectedRoot, err := rlnInstance.GetMerkleRoot()
	s.Require().NoError(err)

	// register the members to the contract
	_, membershipGroupIndex := s.register(credentials1, s.u1PrivKey, "./test_onchain.json")
	_, membershipGroupIndex = s.register(credentials2, s.u1PrivKey, "./test_onchain.json")

	// Creating relay
	port, err := tests.FindFreePort(s.T(), "", 5)
	s.Require().NoError(err)

	host, err := tests.MakeHost(context.Background(), port, rand.Reader)
	s.Require().NoError(err)

	relay := relay.NewWakuRelay(nil, 0, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, utils.Logger())
	relay.SetHost(host)
	err = relay.Start(context.TODO())
	defer relay.Stop()
	s.Require().NoError(err)

	// Subscribing to topic

	sub, err := relay.SubscribeToTopic(context.TODO(), RLNRELAY_PUBSUB_TOPIC)
	s.Require().NoError(err)
	defer sub.Unsubscribe()

	// mount the rln relay protocol in the on-chain/dynamic mode
	// TODO: This assumes the keystoreIndex is 0, but there are two possible credentials in this keystore due to using the same contract address
	// when credentials1 and credentials2 were registered. We should remove this hardcoded value and obtain the correct value when the credentials are persisted
	keystoreIndex := uint(0)
	gm, err := dynamic.NewDynamicGroupManager(s.clientAddr, s.rlnAddr, membershipGroupIndex, "./test_onchain.json", keystorePassword, keystoreIndex, false, utils.Logger())
	s.Require().NoError(err)

	rlnRelay, err := New(relay, gm, "test-merkle-tree.db", RLNRELAY_PUBSUB_TOPIC, RLNRELAY_CONTENT_TOPIC, nil, timesource.NewDefaultClock(), utils.Logger())
	s.Require().NoError(err)

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
	// Creating an RLN instance (just for generating membership keys)
	rlnInstance, err := rln.NewRLN()
	s.Require().NoError(err)

	// Node 1 ============================================================
	port1, err := tests.FindFreePort(s.T(), "", 5)
	s.Require().NoError(err)

	host1, err := tests.MakeHost(context.Background(), port1, rand.Reader)
	s.Require().NoError(err)

	relay1 := relay.NewWakuRelay(nil, 0, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, utils.Logger())
	relay1.SetHost(host1)
	err = relay1.Start(context.TODO())
	defer relay1.Stop()
	s.Require().NoError(err)

	sub1, err := relay1.SubscribeToTopic(context.TODO(), RLNRELAY_PUBSUB_TOPIC)
	s.Require().NoError(err)
	defer sub1.Unsubscribe()

	// Register credentials1 in contract and keystore1
	credentials1 := s.generateCredentials(rlnInstance)
	keystorePath1 := "./test_onchain.json"
	_, membershipGroupIndex := s.register(credentials1, s.u1PrivKey, keystorePath1)
	defer s.removeCredentials(keystorePath1)

	// mount the rln relay protocol in the on-chain/dynamic mode
	gm1, err := dynamic.NewDynamicGroupManager(s.clientAddr, s.rlnAddr, membershipGroupIndex, keystorePath1, keystorePassword, 0, false, utils.Logger())
	s.Require().NoError(err)

	rlnRelay1, err := New(relay1, gm1, "test-correct-registration-1.db", RLNRELAY_PUBSUB_TOPIC, RLNRELAY_CONTENT_TOPIC, nil, timesource.NewDefaultClock(), utils.Logger())
	s.Require().NoError(err)
	err = rlnRelay1.Start(context.TODO())
	s.Require().NoError(err)

	// Node 2 ============================================================
	port2, err := tests.FindFreePort(s.T(), "", 5)
	s.Require().NoError(err)

	host2, err := tests.MakeHost(context.Background(), port2, rand.Reader)
	s.Require().NoError(err)

	relay2 := relay.NewWakuRelay(nil, 0, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, utils.Logger())
	relay2.SetHost(host2)
	err = relay2.Start(context.TODO())
	defer relay2.Stop()
	s.Require().NoError(err)

	sub2, err := relay2.SubscribeToTopic(context.TODO(), RLNRELAY_PUBSUB_TOPIC)
	s.Require().NoError(err)
	defer sub2.Unsubscribe()

	// Register credentials2 in contract and keystore2
	credentials2 := s.generateCredentials(rlnInstance)
	keystorePath2 := "./test_onchain2.json"
	_, membershipGroupIndex = s.register(credentials2, s.u2PrivKey, keystorePath2)
	defer s.removeCredentials(keystorePath2)

	// mount the rln relay protocol in the on-chain/dynamic mode
	gm2, err := dynamic.NewDynamicGroupManager(s.clientAddr, s.rlnAddr, membershipGroupIndex, keystorePath2, keystorePassword, 0, false, utils.Logger())
	s.Require().NoError(err)

	rlnRelay2, err := New(relay2, gm2, "test-correct-registration-2.db", RLNRELAY_PUBSUB_TOPIC, RLNRELAY_CONTENT_TOPIC, nil, timesource.NewDefaultClock(), utils.Logger())
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

	s.Require().Equal(rln.MembershipIndex(0), idx1)
	s.Require().Equal(rln.MembershipIndex(1), idx2)
}
