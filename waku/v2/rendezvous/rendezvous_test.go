package rendezvous

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/persistence/sqlite"
	v2 "github.com/waku-org/go-waku/waku/v2"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

type PeerConn struct {
	ch <-chan v2.PeerData
}

func (p *PeerConn) Subscribe(ctx context.Context, ch <-chan v2.PeerData) {
	p.ch = ch

}

func NewPeerConn() *PeerConn {
	x := PeerConn{}
	return &x
}

const testTopic = "test"

func TestRendezvous(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	port1, err := tests.FindFreePort(t, "", 5)
	require.NoError(t, err)
	host1, err := tests.MakeHost(ctx, port1, rand.Reader)
	require.NoError(t, err)

	var db *sql.DB
	db, migration, err := sqlite.NewDB(":memory:")
	require.NoError(t, err)

	err = migration(db)
	require.NoError(t, err)

	rdb := NewDB(ctx, db, utils.Logger())
	rendezvousPoint := NewRendezvous(rdb, nil, utils.Logger())
	rendezvousPoint.SetHost(host1)
	err = rendezvousPoint.Start(ctx)
	require.NoError(t, err)
	defer rendezvousPoint.Stop()
	host1RP := NewRendezvousPoint(host1.ID())

	hostInfo, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/p2p/%s", host1.ID().Pretty()))
	host1Addr := host1.Addrs()[0].Encapsulate(hostInfo)

	port2, err := tests.FindFreePort(t, "", 5)
	require.NoError(t, err)
	host2, err := tests.MakeHost(ctx, port2, rand.Reader)
	require.NoError(t, err)

	info, err := peer.AddrInfoFromP2pAddr(host1Addr)
	require.NoError(t, err)

	host2.Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.PermanentAddrTTL)
	err = host2.Peerstore().AddProtocols(info.ID, RendezvousID)
	require.NoError(t, err)

	rendezvousClient1 := NewRendezvous(nil, nil, utils.Logger())
	rendezvousClient1.SetHost(host2)

	rendezvousClient1.RegisterWithNamespace(context.Background(), testTopic, []*RendezvousPoint{host1RP})

	port3, err := tests.FindFreePort(t, "", 5)
	require.NoError(t, err)
	host3, err := tests.MakeHost(context.Background(), port3, rand.Reader)
	require.NoError(t, err)

	host3.Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.PermanentAddrTTL)
	err = host3.Peerstore().AddProtocols(info.ID, RendezvousID)
	require.NoError(t, err)

	myPeerConnector := NewPeerConn()

	rendezvousClient2 := NewRendezvous(nil, myPeerConnector, utils.Logger())
	rendezvousClient2.SetHost(host3)

	timedCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()

	go rendezvousClient2.DiscoverWithNamespace(timedCtx, testTopic, host1RP, 1)
	time.Sleep(500 * time.Millisecond)

	timer := time.After(3 * time.Second)
	select {
	case <-timer:
		require.Fail(t, "no peer discovered")
	case p := <-myPeerConnector.ch:
		require.Equal(t, p.AddrInfo.ID.Pretty(), host2.ID().Pretty())
	}
}
