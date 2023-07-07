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
	port1, err := tests.FindFreePort(t, "", 5)
	require.NoError(t, err)
	host1, err := tests.MakeHost(context.Background(), port1, rand.Reader)
	require.NoError(t, err)

	var db *sql.DB
	db, migration, err := sqlite.NewDB(":memory:")
	require.NoError(t, err)

	err = migration(db)
	require.NoError(t, err)

	rdb := NewDB(context.Background(), db, utils.Logger())
	rendezvousPoint := NewRendezvous(true, rdb, nil, NewPeerConn(), utils.Logger())
	rendezvousPoint.SetHost(host1)
	err = rendezvousPoint.Start(context.Background())
	require.NoError(t, err)
	defer rendezvousPoint.Stop()

	hostInfo, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/p2p/%s", host1.ID().Pretty()))
	host1Addr := host1.Addrs()[0].Encapsulate(hostInfo)

	port2, err := tests.FindFreePort(t, "", 5)
	require.NoError(t, err)
	host2, err := tests.MakeHost(context.Background(), port2, rand.Reader)
	require.NoError(t, err)

	info, err := peer.AddrInfoFromP2pAddr(host1Addr)
	require.NoError(t, err)

	host2.Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.PermanentAddrTTL)
	err = host2.Peerstore().AddProtocols(info.ID, RendezvousID)
	require.NoError(t, err)

	rendezvousClient1 := NewRendezvous(false, nil, []peer.ID{host1.ID()}, NewPeerConn(), utils.Logger())
	rendezvousClient1.SetHost(host2)
	err = rendezvousClient1.Start(context.Background())
	require.NoError(t, err)
	defer rendezvousClient1.Stop()

	rendezvousClient1.Register(context.Background(), testTopic)

	port3, err := tests.FindFreePort(t, "", 5)
	require.NoError(t, err)
	host3, err := tests.MakeHost(context.Background(), port3, rand.Reader)
	require.NoError(t, err)

	host3.Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.PermanentAddrTTL)
	err = host3.Peerstore().AddProtocols(info.ID, RendezvousID)
	require.NoError(t, err)

	myPeerConnector := NewPeerConn()
	rendezvousClient2 := NewRendezvous(false, nil, []peer.ID{host1.ID()}, myPeerConnector, utils.Logger())
	rendezvousClient2.SetHost(host3)
	err = rendezvousClient2.Start(context.Background())
	require.NoError(t, err)
	defer rendezvousClient2.Stop()

	timedCtx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	go rendezvousClient2.Discover(timedCtx, testTopic, 5)
	time.Sleep(500 * time.Millisecond)

	timer := time.After(3 * time.Second)
	select {
	case <-timer:
		require.Fail(t, "no peer discovered")
	case p := <-myPeerConnector.ch:
		require.Equal(t, p.AddrInfo.ID.Pretty(), host2.ID().Pretty())
	}
}
