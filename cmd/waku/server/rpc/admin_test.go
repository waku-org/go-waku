package rpc

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/multiformats/go-multiaddr"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/timesource"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

func makeAdminService(t *testing.T) *AdminService {
	options := node.WithWakuRelay()
	n, err := node.New(options)
	require.NoError(t, err)
	err = n.Start(context.Background())
	require.NoError(t, err)
	return &AdminService{n, utils.Logger()}
}

func TestV1Peers(t *testing.T) {
	port, err := tests.FindFreePort(t, "", 5)
	require.NoError(t, err)

	broadcaster := relay.NewBroadcaster(10)
	require.NoError(t, broadcaster.Start(context.Background()))

	host, err := tests.MakeHost(context.Background(), port, rand.Reader)
	require.NoError(t, err)
	relay := relay.NewWakuRelay(broadcaster, 0, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, utils.Logger())
	relay.SetHost(host)
	err = relay.Start(context.Background())
	require.NoError(t, err)
	defer relay.Stop()

	var reply PeersReply

	request, err := http.NewRequest(http.MethodPost, "url", bytes.NewReader([]byte("")))
	require.NoError(t, err)

	a := makeAdminService(t)

	err = a.GetV1Peers(request, &GetPeersArgs{}, &reply)
	require.NoError(t, err)
	require.Len(t, reply, 0)

	var reply2 SuccessReply

	hostInfo, err := multiaddr.NewMultiaddr(fmt.Sprintf("/p2p/%s", host.ID().String()))
	require.NoError(t, err)

	var addr multiaddr.Multiaddr
	for _, a := range host.Addrs() {
		addr = a.Encapsulate(hostInfo)
		break
	}
	err = a.PostV1Peers(request, &PeersArgs{Peers: []string{addr.String()}}, &reply2)
	require.NoError(t, err)
	require.True(t, reply2)

	time.Sleep(2 * time.Second)

	err = a.GetV1Peers(request, &GetPeersArgs{}, &reply)
	require.NoError(t, err)
	require.Len(t, reply, 1)
}
