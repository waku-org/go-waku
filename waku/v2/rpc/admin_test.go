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

	"github.com/status-im/go-waku/tests"
	"github.com/status-im/go-waku/waku/v2/node"
	"github.com/status-im/go-waku/waku/v2/protocol/relay"
	"github.com/status-im/go-waku/waku/v2/utils"
	"github.com/stretchr/testify/require"
)

func makeAdminService(t *testing.T) *AdminService {
	options := node.WithWakuRelay()
	n, err := node.New(context.Background(), options)
	require.NoError(t, err)
	err = n.Start()
	require.NoError(t, err)
	return &AdminService{n, utils.Logger()}
}

func TestV1Peers(t *testing.T) {
	port, err := tests.FindFreePort(t, "", 5)
	require.NoError(t, err)

	host, err := tests.MakeHost(context.Background(), port, rand.Reader)
	require.NoError(t, err)
	relay, err := relay.NewWakuRelay(context.Background(), host, nil, 0, utils.Logger())
	require.NoError(t, err)
	defer relay.Stop()

	var reply PeersReply

	request, err := http.NewRequest(http.MethodPost, "url", bytes.NewReader([]byte("")))
	require.NoError(t, err)

	a := makeAdminService(t)

	err = a.GetV1Peers(request, &GetPeersArgs{}, &reply)
	require.NoError(t, err)
	require.Len(t, reply.Peers, 0)

	var reply2 SuccessReply

	hostInfo, err := multiaddr.NewMultiaddr(fmt.Sprintf("/p2p/%s", host.ID().Pretty()))
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
	require.Len(t, reply.Peers, 2)
}
