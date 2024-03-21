package node

import (
	"bytes"
	"context"
	"math/big"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/persistence"
	"github.com/waku-org/go-waku/waku/persistence/sqlite"
	"github.com/waku-org/go-waku/waku/v2/dnsdisc"
	"github.com/waku-org/go-waku/waku/v2/peerstore"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/filter"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/protocol/store"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

func createTestMsg(version uint32) *pb.WakuMessage {
	message := new(pb.WakuMessage)
	message.Payload = []byte{0, 1, 2}
	message.Version = proto.Uint32(version)
	message.Timestamp = proto.Int64(123456)
	message.ContentTopic = "abc"
	return message
}

func TestWakuNode2(t *testing.T) {
	hostAddr, _ := net.ResolveTCPAddr("tcp", "0.0.0.0:0")

	key, err := tests.RandomHex(32)
	require.NoError(t, err)
	prvKey, err := crypto.HexToECDSA(key)
	require.NoError(t, err)

	ctx := context.Background()

	wakuNode, err := New(
		WithPrivateKey(prvKey),
		WithHostAddress(hostAddr),
		WithWakuRelay(),
	)
	require.NoError(t, err)

	err = wakuNode.Start(ctx)
	require.NoError(t, err)

	_, err = wakuNode.Relay().Subscribe(ctx, protocol.NewContentFilter("waku/rs/1/1"))
	require.NoError(t, err)
	time.Sleep(time.Second * 1)

	err = wakuNode.Relay().Unsubscribe(ctx, protocol.NewContentFilter("waku/rs/1/1"))
	require.NoError(t, err)

	defer wakuNode.Stop()
}

func int2Bytes(i int) []byte {
	if i > 0 {
		return append(big.NewInt(int64(i)).Bytes(), byte(1))
	}
	return append(big.NewInt(int64(i)).Bytes(), byte(0))
}

func TestUpAndDown(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	hostAddr1, _ := net.ResolveTCPAddr("tcp", "0.0.0.0:0")
	key1, _ := tests.RandomHex(32)
	prvKey1, _ := crypto.HexToECDSA(key1)

	nodes, err := dnsdisc.RetrieveNodes(context.Background(), "enrtree://AIRVQ5DDA4FFWLRBCHJWUWOO6X6S4ZTZ5B667LQ6AJU6PEYDLRD5O@sandbox.waku.nodes.status.im")
	require.NoError(t, err)

	var bootnodes []*enode.Node
	for _, n := range nodes {
		if n.ENR != nil {
			bootnodes = append(bootnodes, n.ENR)
		}
	}

	wakuNode1, err := New(
		WithPrivateKey(prvKey1),
		WithHostAddress(hostAddr1),
		WithWakuRelay(),
		WithDiscoveryV5(0, bootnodes, true),
	)
	require.NoError(t, err)

	for i := 0; i < 5; i++ {
		utils.Logger().Info("Starting...", zap.Int("iteration", i))
		err = wakuNode1.Start(ctx)
		require.NoError(t, err)
		err = wakuNode1.DiscV5().Start(ctx)
		require.NoError(t, err)
		utils.Logger().Info("Started!", zap.Int("iteration", i))
		time.Sleep(3 * time.Second)
		utils.Logger().Info("Stopping...", zap.Int("iteration", i))
		wakuNode1.Stop()
		utils.Logger().Info("Stopped!", zap.Int("iteration", i))
	}
}

func Test500(t *testing.T) {
	maxMsgs := 500
	maxMsgBytes := int2Bytes(maxMsgs)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	hostAddr1, _ := net.ResolveTCPAddr("tcp", "0.0.0.0:0")
	key1, _ := tests.RandomHex(32)
	prvKey1, _ := crypto.HexToECDSA(key1)

	hostAddr2, _ := net.ResolveTCPAddr("tcp", "0.0.0.0:0")
	key2, _ := tests.RandomHex(32)
	prvKey2, _ := crypto.HexToECDSA(key2)

	wakuNode1, err := New(
		WithPrivateKey(prvKey1),
		WithHostAddress(hostAddr1),
		WithWakuRelay(),
	)
	require.NoError(t, err)
	err = wakuNode1.Start(ctx)
	require.NoError(t, err)
	defer wakuNode1.Stop()

	wakuNode2, err := New(
		WithPrivateKey(prvKey2),
		WithHostAddress(hostAddr2),
		WithWakuRelay(),
	)
	require.NoError(t, err)
	err = wakuNode2.Start(ctx)
	require.NoError(t, err)
	defer wakuNode2.Stop()

	err = wakuNode2.DialPeer(ctx, wakuNode1.ListenAddresses()[0].String())
	require.NoError(t, err)

	time.Sleep(2 * time.Second)

	sub1, err := wakuNode1.Relay().Subscribe(ctx, protocol.NewContentFilter(relay.DefaultWakuTopic))
	require.NoError(t, err)
	sub2, err := wakuNode1.Relay().Subscribe(ctx, protocol.NewContentFilter(relay.DefaultWakuTopic))
	require.NoError(t, err)

	wg := sync.WaitGroup{}
	wg.Add(3)
	go func() {
		defer wg.Done()

		ticker := time.NewTimer(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				require.Fail(t, "Timeout Sub1")
			case msg := <-sub1[0].Ch:
				if msg == nil {
					return
				}
				if bytes.Equal(msg.Message().Payload, maxMsgBytes) {
					return
				}
			}
		}
	}()

	go func() {
		defer wg.Done()

		ticker := time.NewTimer(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				require.Fail(t, "Timeout Sub2")
			case msg := <-sub2[0].Ch:
				if msg == nil {
					return
				}
				if bytes.Equal(msg.Message().Payload, maxMsgBytes) {
					return
				}
			}
		}
	}()

	go func() {
		defer wg.Done()
		for i := 1; i <= maxMsgs; i++ {
			msg := createTestMsg(0)
			msg.Payload = int2Bytes(i)
			msg.Timestamp = proto.Int64(int64(i))
			if _, err := wakuNode2.Relay().Publish(ctx, msg, relay.WithDefaultPubsubTopic()); err != nil {
				require.Fail(t, "Could not publish all messages")
			}
			time.Sleep(5 * time.Millisecond)
		}
	}()

	wg.Wait()

}

func TestDecoupledStoreFromRelay(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// NODE1: Relay Node + Filter Server
	hostAddr1, err := net.ResolveTCPAddr("tcp", "0.0.0.0:0")
	require.NoError(t, err)
	wakuNode1, err := New(
		WithHostAddress(hostAddr1),
		WithWakuRelay(),
		WithWakuFilterFullNode(),
	)
	require.NoError(t, err)
	err = wakuNode1.Start(ctx)
	require.NoError(t, err)
	defer wakuNode1.Stop()

	subs, err := wakuNode1.Relay().Subscribe(ctx, protocol.NewContentFilter(relay.DefaultWakuTopic))
	require.NoError(t, err)
	defer subs[0].Unsubscribe()

	// NODE2: Filter Client/Store
	db, err := sqlite.NewDB(":memory:", utils.Logger())
	require.NoError(t, err)
	dbStore, err := persistence.NewDBStore(prometheus.DefaultRegisterer, utils.Logger(), persistence.WithDB(db), persistence.WithMigrations(sqlite.Migrations))
	require.NoError(t, err)

	hostAddr2, err := net.ResolveTCPAddr("tcp", "0.0.0.0:0")
	require.NoError(t, err)
	wakuNode2, err := New(
		WithHostAddress(hostAddr2),
		WithWakuFilterLightNode(),
		WithWakuStore(),
		WithMessageProvider(dbStore),
	)
	require.NoError(t, err)
	err = wakuNode2.Start(ctx)
	require.NoError(t, err)
	defer wakuNode2.Stop()

	peerID, err := wakuNode2.AddPeer(wakuNode1.ListenAddresses()[0], peerstore.Static, []string{relay.DefaultWakuTopic}, filter.FilterSubscribeID_v20beta1)
	require.NoError(t, err)

	subscription, err := wakuNode2.FilterLightnode().Subscribe(ctx, protocol.ContentFilter{
		PubsubTopic:   relay.DefaultWakuTopic,
		ContentTopics: protocol.NewContentTopicSet("abc"),
	}, filter.WithPeer(peerID))
	require.NoError(t, err)

	// Sleep to make sure the filter is subscribed
	time.Sleep(1 * time.Second)

	// Send MSG1 on NODE1
	msg := createTestMsg(0)
	msg.Payload = []byte{1, 2, 3, 4, 5}
	msg.Timestamp = utils.GetUnixEpoch()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		// MSG1 should be pushed in NODE2 via filter
		defer wg.Done()
		env, ok := <-subscription[0].C
		if !ok {
			require.Fail(t, "no message")
		}
		require.Equal(t, msg.Timestamp, env.Message().Timestamp)
	}()

	time.Sleep(500 * time.Millisecond)

	if _, err := wakuNode1.Relay().Publish(ctx, msg, relay.WithDefaultPubsubTopic()); err != nil {
		require.Fail(t, "Could not publish all messages")
	}

	wg.Wait()

	// NODE3: Query from NODE2
	hostAddr3, err := net.ResolveTCPAddr("tcp", "0.0.0.0:0")
	require.NoError(t, err)
	wakuNode3, err := New(
		WithHostAddress(hostAddr3),
		WithWakuFilterLightNode(),
	)
	require.NoError(t, err)
	err = wakuNode3.Start(ctx)
	require.NoError(t, err)
	defer wakuNode3.Stop()

	_, err = wakuNode3.AddPeer(wakuNode2.ListenAddresses()[0], peerstore.Static, []string{relay.DefaultWakuTopic}, store.StoreID_v20beta4)
	require.NoError(t, err)
	time.Sleep(2 * time.Second)
	// NODE2 should have returned the message received via filter
	result, err := wakuNode3.Store().Query(ctx, store.Query{})
	require.NoError(t, err)
	require.Len(t, result.Messages, 1)
	require.Equal(t, msg.Timestamp, result.Messages[0].Timestamp)
}

func TestStaticShardingMultipleTopics(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Node1 with Relay
	hostAddr1, err := net.ResolveTCPAddr("tcp", "0.0.0.0:0")
	require.NoError(t, err)
	wakuNode1, err := New(
		WithHostAddress(hostAddr1),
		WithWakuRelay(),
	)
	require.NoError(t, err)
	err = wakuNode1.Start(ctx)
	require.NoError(t, err)
	defer wakuNode1.Stop()

	pubSubTopic1 := protocol.NewStaticShardingPubsubTopic(uint16(0), uint16(0))
	pubSubTopic1Str := pubSubTopic1.String()
	contentTopic1 := "/test/2/my-app/sharded"

	pubSubTopic2 := protocol.NewStaticShardingPubsubTopic(uint16(0), uint16(10))
	pubSubTopic2Str := pubSubTopic2.String()
	contentTopic2 := "/test/3/my-app/sharded"

	r := wakuNode1.Relay()

	subs1, err := r.Subscribe(ctx, protocol.NewContentFilter(pubSubTopic1Str, contentTopic1))
	require.NoError(t, err)

	subs2, err := r.Subscribe(ctx, protocol.NewContentFilter(pubSubTopic2Str, contentTopic2))
	require.NoError(t, err)

	require.NotEqual(t, subs1[0].ID, subs2[0].ID)

	require.True(t, r.IsSubscribed(pubSubTopic1Str))
	require.True(t, r.IsSubscribed(pubSubTopic2Str))

	s1, err := r.GetSubscription(contentTopic1)
	require.NoError(t, err)
	s2, err := r.GetSubscription(contentTopic2)
	require.NoError(t, err)
	require.Equal(t, s1.ID, subs1[0].ID)
	require.Equal(t, s2.ID, subs2[0].ID)

}
