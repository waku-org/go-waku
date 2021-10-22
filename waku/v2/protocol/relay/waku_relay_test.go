package relay

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"net"
	"testing"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/multiformats/go-multiaddr"
	"github.com/status-im/go-waku/waku/v2/protocol/pb"
	"github.com/stretchr/testify/require"
)

func findFreePort(t *testing.T, host string, maxAttempts int) (int, error) {
	t.Helper()

	if host == "" {
		host = "localhost"
	}

	for i := 0; i < maxAttempts; i++ {
		addr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort(host, "0"))
		if err != nil {
			t.Logf("unable to resolve tcp addr: %v", err)
			continue
		}
		l, err := net.ListenTCP("tcp", addr)
		if err != nil {
			l.Close()
			t.Logf("unable to listen on addr %q: %v", addr, err)
			continue
		}

		port := l.Addr().(*net.TCPAddr).Port
		l.Close()
		return port, nil

	}

	return 0, fmt.Errorf("no free port found")
}

func makeHost(ctx context.Context, port int, randomness io.Reader) (host.Host, error) {
	// Creates a new RSA key pair for this host.
	prvKey, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, randomness)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	// 0.0.0.0 will listen on any interface device.
	sourceMultiAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port))

	// libp2p.New constructs a new libp2p Host.
	// Other options can be added here.
	return libp2p.New(
		ctx,
		libp2p.ListenAddrs(sourceMultiAddr),
		libp2p.Identity(prvKey),
	)
}

func TestWakuRelay(t *testing.T) {
	var testTopic Topic = "/waku/2/go/relay/test"

	port, err := findFreePort(t, "", 5)
	require.NoError(t, err)

	host, err := makeHost(context.Background(), port, rand.Reader)
	require.NoError(t, err)

	relay, err := NewWakuRelay(context.Background(), host)
	defer relay.Stop()
	require.NoError(t, err)

	sub, isNew, err := relay.Subscribe(testTopic)
	defer sub.Cancel()
	require.NoError(t, err)
	require.True(t, isNew)

	_, isNew, err = relay.Subscribe(testTopic)
	require.NoError(t, err)
	require.False(t, isNew)

	topics := relay.Topics()
	require.Equal(t, 1, len(topics))
	require.Equal(t, testTopic, topics[0])

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		defer cancel()

		_, err := sub.Next(ctx)
		require.NoError(t, err)
	}()

	msg := &pb.WakuMessage{
		Payload:      []byte{1},
		Version:      0,
		ContentTopic: "test",
		Timestamp:    0,
	}
	_, err = relay.Publish(context.Background(), msg, &testTopic)
	require.NoError(t, err)

	<-ctx.Done()
}
