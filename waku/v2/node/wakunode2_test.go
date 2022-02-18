package node

import (
	"context"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/status-im/go-waku/tests"
	"github.com/stretchr/testify/require"
)

func TestWakuNode2(t *testing.T) {
	hostAddr, _ := net.ResolveTCPAddr("tcp", "0.0.0.0:0")

	key, err := tests.RandomHex(32)
	require.NoError(t, err)
	prvKey, err := crypto.HexToECDSA(key)
	require.NoError(t, err)

	ctx := context.Background()

	wakuNode, err := New(ctx,
		WithPrivateKey(prvKey),
		WithHostAddress(hostAddr),
		WithWakuRelay(),
	)
	require.NoError(t, err)

	err = wakuNode.Start()
	defer wakuNode.Stop()

	require.NoError(t, err)
}

func Test1100(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	hostAddr1, _ := net.ResolveTCPAddr("tcp", "0.0.0.0:0")
	key1, _ := tests.RandomHex(32)
	prvKey1, _ := crypto.HexToECDSA(key1)

	hostAddr2, _ := net.ResolveTCPAddr("tcp", "0.0.0.0:0")
	key2, _ := tests.RandomHex(32)
	prvKey2, _ := crypto.HexToECDSA(key2)

	wakuNode1, err := New(ctx,
		WithPrivateKey(prvKey1),
		WithHostAddress(hostAddr1),
		WithWakuRelay(),
	)
	require.NoError(t, err)
	err = wakuNode1.Start()
	require.NoError(t, err)
	defer wakuNode1.Stop()

	wakuNode2, err := New(ctx,
		WithPrivateKey(prvKey2),
		WithHostAddress(hostAddr2),
		WithWakuRelay(),
	)
	require.NoError(t, err)
	err = wakuNode2.Start()
	require.NoError(t, err)
	defer wakuNode2.Stop()

	err = wakuNode2.DialPeer(ctx, wakuNode1.ListenAddresses()[0].String())
	require.NoError(t, err)

	time.Sleep(2 * time.Second)

	sub1, err := wakuNode1.Relay().Subscribe(ctx)
	require.NoError(t, err)
	sub2, err := wakuNode1.Relay().Subscribe(ctx)
	require.NoError(t, err)

	wg := sync.WaitGroup{}
	wg.Add(3)
	go func() {
		defer wg.Done()

		ticker := time.NewTimer(20 * time.Second)
		defer ticker.Stop()

		msgCnt := 0
		for {
			select {
			case <-ticker.C:
				if msgCnt != 1100 {
					require.Fail(t, "Timeout Sub1")
				}
			case <-sub1.C:
				msgCnt++
				if msgCnt == 1100 {
					return
				}
			}
		}
	}()

	go func() {
		defer wg.Done()

		ticker := time.NewTimer(20 * time.Second)
		defer ticker.Stop()

		msgCnt := 0
		for {
			select {
			case <-ticker.C:
				if msgCnt != 1100 {
					require.Fail(t, "Timeout Sub2")
				}
			case <-sub2.C:
				msgCnt++
				if msgCnt == 1100 {
					return
				}
			}
		}
	}()

	go func() {
		defer wg.Done()
		for i := 1; i <= 1100; i++ {
			msg := createTestMsg(0)
			msg.Payload = []byte(fmt.Sprint(i))
			msg.Timestamp = float64(i)
			if err := wakuNode2.Publish(ctx, msg); err != nil {
				require.Fail(t, "Could not publish all messages")
			}
		}
	}()

	wg.Wait()

}
