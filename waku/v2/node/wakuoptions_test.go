package node

import (
	"net"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/persistence"
	"github.com/waku-org/go-waku/waku/v2/protocol/store"
)

func TestWakuOptions(t *testing.T) {
	connStatusChan := make(chan ConnStatus, 100)

	key, err := tests.RandomHex(32)
	require.NoError(t, err)

	prvKey, err := crypto.HexToECDSA(key)
	require.NoError(t, err)

	hostAddr, _ := net.ResolveTCPAddr("tcp", "0.0.0.0:0")

	addr, err := multiaddr.NewMultiaddr("/ip4/0.0.0.0/tcp/4000/ws")
	require.NoError(t, err)

	storeFactory := func(w *WakuNode) store.Store {
		return store.NewWakuStore(w.opts.messageProvider, w.timesource, w.log)
	}

	options := []WakuNodeOption{
		WithHostAddress(hostAddr),
		WithAdvertiseAddresses(addr),
		WithMultiaddress([]multiaddr.Multiaddr{addr}),
		WithPrivateKey(prvKey),
		WithLibP2POptions(),
		WithWakuRelay(),
		WithLegacyWakuFilter(true),
		WithDiscoveryV5(123, nil, false),
		WithWakuStore(),
		WithMessageProvider(&persistence.DBStore{}),
		WithLightPush(),
		WithKeepAlive(time.Hour),
		WithConnectionStatusChannel(connStatusChan),
		WithWakuStoreFactory(storeFactory),
	}

	params := new(WakuNodeParameters)

	for _, opt := range options {
		require.NoError(t, opt(params))
	}

	require.NotNil(t, params.multiAddr)
	require.NotNil(t, params.privKey)
	require.NotNil(t, params.connStatusC)
}
