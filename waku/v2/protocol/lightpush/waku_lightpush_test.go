package lightpush

import (
	"context"
	"crypto/rand"
	"fmt"
	"sync"
	"testing"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peerstore"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/status-im/go-waku/tests"
	"github.com/status-im/go-waku/waku/v2/protocol/pb"
	"github.com/status-im/go-waku/waku/v2/protocol/relay"
	"github.com/stretchr/testify/require"
)

// Node1:   Relay
// Node2: Relay+Lightpush
// Node3: Client that will lightpush a message

// Node1  and Node 2 must be peers
// Node 3 and Node 2 must be peers

// Test would be like this:

// Node 3 will use lightpush request, sending the message to Node2
// Node2 should then broadcast the message and not fail
// Node1 should receive the message in relay
// Node3 should receive a succesful response from lightpushRequest

func makeWakuRelay(t *testing.T, topic relay.Topic) (*relay.WakuRelay, *pubsub.Subscription, host.Host) {
	port, err := tests.FindFreePort(t, "", 5)
	require.NoError(t, err)

	host, err := tests.MakeHost(context.Background(), port, rand.Reader)
	require.NoError(t, err)

	relay, err := relay.NewWakuRelay(context.Background(), host)
	require.NoError(t, err)

	sub, _, err := relay.Subscribe(topic)
	require.NoError(t, err)

	return relay, sub, host
}

func TestWakuLightPush(t *testing.T) {
	// protocol := libp2pProtocol.ID("/vac/waku/store/2.0.0-beta3")
	var testTopic relay.Topic = "/waku/2/go/lightpush/test"
	node1, sub1, host1 := makeWakuRelay(t, testTopic)
	defer node1.Stop()
	defer sub1.Cancel()

	node2, sub2, host2 := makeWakuRelay(t, testTopic)
	defer node2.Stop()
	defer sub2.Cancel()

	ctx := context.Background()
	lightPushNode2 := NewWakuLightPush(ctx, host2, node2)
	defer lightPushNode2.Stop()

	port, err := tests.FindFreePort(t, "", 5)
	require.NoError(t, err)

	clientHost, err := tests.MakeHost(context.Background(), port, rand.Reader)
	require.NoError(t, err)
	client := NewWakuLightPush(ctx, clientHost, nil)

	host2.Peerstore().AddAddr(host1.ID(), tests.GetHostAddress(host1), peerstore.PermanentAddrTTL)
	// err = host2.Peerstore().AddProtocols(host1.ID(), string(rendezvous.RendezvousID_v001))
	require.NoError(t, err)

	fmt.Println(" begin")
	fmt.Println(host1.ID())
	fmt.Println(host2.ID())
	fmt.Println(clientHost.ID())
	clientHost.Peerstore().AddAddr(host2.ID(), tests.GetHostAddress(host2), peerstore.PermanentAddrTTL)
	err = clientHost.Peerstore().AddProtocols(host2.ID(), string(LightPushID_v20beta1))
	require.NoError(t, err)

	req := new(pb.PushRequest)
	req.Message = &pb.WakuMessage{
		Payload:      []byte{1},
		Version:      0,
		ContentTopic: "test",
		Timestamp:    0,
	}
	req.PubsubTopic = string(testTopic)
	fmt.Println(req.PubsubTopic)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := sub1.Next(context.Background())
		require.NoError(t, err)
		fmt.Println(" hello111")
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := sub2.Next(context.Background())
		require.NoError(t, err)
		fmt.Println(" hello2222")
	}()

	fmt.Println(" request")
	resp, err := client.Request(ctx, req, []LightPushOption{}...)
	require.NoError(t, err)
	require.True(t, resp.IsSuccess)

	wg.Wait()
}
