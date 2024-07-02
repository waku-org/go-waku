package node

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
	"github.com/waku-org/go-waku/logging"
	"go.uber.org/zap"
)

const maxAllowedPingFailures = 2

// If the difference between the last time the keep alive code was executed and now is greater
// than sleepDectectionIntervalFactor * keepAlivePeriod, force the ping verification to disconnect
// the peers if they don't reply back
const sleepDetectionIntervalFactor = 3

const maxPeersToPing = 10

// startKeepAlive creates a go routine that periodically pings connected peers.
// This is necessary because TCP connections are automatically closed due to inactivity,
// and doing a ping will avoid this (with a small bandwidth cost)
func (w *WakuNode) startKeepAlive(ctx context.Context, randomPeersPingDuration time.Duration, allPeersPingDuration time.Duration) {
	defer w.wg.Done()
	w.log.Info("setting up ping protocol", zap.Duration("randomPeersPingDuration", randomPeersPingDuration), zap.Duration("allPeersPingDuration", allPeersPingDuration))

	randomPeersTicker := time.NewTicker(randomPeersPingDuration)
	defer randomPeersTicker.Stop()

	allPeersTicker := time.NewTicker(randomPeersPingDuration)
	defer allPeersTicker.Stop()

	lastTimeExecuted := w.timesource.Now()

	sleepDetectionInterval := int64(randomPeersPingDuration) * sleepDetectionIntervalFactor

	for {
		peersToPing := []peer.ID{}

		select {
		case <-allPeersTicker.C:
			peersToPing = w.host.Network().Peers()

		case <-randomPeersTicker.C:
			difference := w.timesource.Now().UnixNano() - lastTimeExecuted.UnixNano()
			if difference > sleepDetectionInterval {
				lastTimeExecuted = w.timesource.Now()
				w.log.Warn("keep alive hasnt been executed recently. Killing all connections")
				for _, p := range w.host.Network().Peers() {
					err := w.host.Network().ClosePeer(p)
					if err != nil {
						w.log.Debug("closing conn to peer", zap.Error(err))
					}
				}
				continue
			}

			if maxPeersToPing-len(peersToPing) > 0 {
				connectedPeers := w.host.Network().Peers()
				rand.Shuffle(len(connectedPeers), func(i, j int) { connectedPeers[i], connectedPeers[j] = connectedPeers[j], connectedPeers[i] })
				peerLen := len(peersToPing)
				if peerLen > maxPeersToPing {
					peerLen = maxPeersToPing
				}
				peersToPing = connectedPeers[0:peerLen]
			}

		case <-ctx.Done():
			w.log.Info("stopping ping protocol")
			return
		}

		pingWg := sync.WaitGroup{}
		pingWg.Add(len(peersToPing))
		for _, p := range peersToPing {
			go w.pingPeer(ctx, &pingWg, p)
		}
		pingWg.Wait()

		lastTimeExecuted = w.timesource.Now()
	}
}

func (w *WakuNode) pingPeer(ctx context.Context, wg *sync.WaitGroup, peerID peer.ID) {
	defer wg.Done()

	logger := w.log.With(logging.HostID("peer", peerID))

	for i := 0; i < maxAllowedPingFailures; i++ {
		if w.host.Network().Connectedness(peerID) != network.Connected {
			// Peer is no longer connected. No need to ping
			return
		}

		logger.Debug("pinging")
		func() {
			ctx, cancel := context.WithTimeout(ctx, 7*time.Second)
			defer cancel()

			pr := ping.Ping(ctx, w.host, peerID)
			select {
			case res := <-pr:
				if res.Error != nil {
					logger.Debug("could not ping", zap.Error(res.Error))
				} else {
					return
				}
			case <-ctx.Done():
				if errors.Is(ctx.Err(), context.Canceled) {
					// Parent context was cancelled
					return
				}

				logger.Debug("could not ping (context)", zap.Error(ctx.Err()))
			}
		}()

	}

	logger.Info("disconnecting dead peer")
	if err := w.host.Network().ClosePeer(peerID); err != nil {
		logger.Debug("closing conn to peer", zap.Error(err))
	}
}
