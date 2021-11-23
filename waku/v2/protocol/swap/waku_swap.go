package swap

import (
	"sync"

	logging "github.com/ipfs/go-log"

	"github.com/libp2p/go-libp2p-core/protocol"
)

const (
	SoftMode int = 0
	MockMode int = 1
	HardMode int = 2
)

const WakuSwapID_v200 = protocol.ID("/vac/waku/swap/2.0.0-beta1")

var log = logging.Logger("wakuswap")

type WakuSwap struct {
	params *SwapParameters

	Accounting      map[string]int
	accountingMutex sync.RWMutex
}

func NewWakuSwap(opts ...SwapOption) *WakuSwap {
	params := &SwapParameters{}

	optList := DefaultOptions()
	optList = append(optList, opts...)
	for _, opt := range optList {
		opt(params)
	}

	return &WakuSwap{
		params:     params,
		Accounting: make(map[string]int),
	}
}

func (s *WakuSwap) sendCheque(peerId string) {
	log.Debug("not yet implemented")
}

func (s *WakuSwap) applyPolicy(peerId string) {
	if s.Accounting[peerId] <= s.params.disconnectThreshold {
		log.Warnf("Disconnect threshhold has been reached for %s at %d", peerId, s.Accounting[peerId])
	}

	if s.Accounting[peerId] >= s.params.paymentThreshold {
		log.Warnf("Disconnect threshhold has been reached for %s at %d", peerId, s.Accounting[peerId])
		if s.params.mode != HardMode {
			s.sendCheque(peerId)
		}
	}
}

func (s *WakuSwap) Credit(peerId string, n int) {
	s.accountingMutex.Lock()
	defer s.accountingMutex.Unlock()

	s.Accounting[peerId] -= n
	s.applyPolicy(peerId)
}

func (s *WakuSwap) Debit(peerId string, n int) {
	s.accountingMutex.Lock()
	defer s.accountingMutex.Unlock()

	s.Accounting[peerId] += n
	s.applyPolicy(peerId)
}
