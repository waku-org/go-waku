package missing

// test

import (
	"context"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/waku-org/go-waku/logging"
	"github.com/waku-org/go-waku/waku/v2/api/common"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	storepb "github.com/waku-org/go-waku/waku/v2/protocol/store/pb"
	"github.com/waku-org/go-waku/waku/v2/timesource"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

const maxContentTopicsPerRequest = 10
const maxMsgHashesPerRequest = 50
const messageFetchPageSize = 100

// MessageTracker should keep track of messages it has seen before and
// provide a way to determine whether a message exists or not. This
// is application specific
type MessageTracker interface {
	MessageExists(pb.MessageHash) (bool, error)
}

// MissingMessageVerifier is used to periodically retrieve missing messages from store nodes that have some specific criteria
type MissingMessageVerifier struct {
	ctx    context.Context
	cancel context.CancelFunc
	params missingMessageVerifierParams

	storenodeRequestor common.StorenodeRequestor
	messageTracker     MessageTracker

	criteriaInterest   map[string]*criteriaInterest // Track message verification requests and when was the last time a pubsub topic was verified for missing messages
	criteriaInterestMu sync.RWMutex

	C chan *protocol.Envelope

	timesource   timesource.Timesource
	logger       *zap.Logger
	isRunning    bool
	runningMutex sync.RWMutex
}

// NewMissingMessageVerifier creates an instance of a MissingMessageVerifier
func NewMissingMessageVerifier(storenodeRequester common.StorenodeRequestor, messageTracker MessageTracker, timesource timesource.Timesource, logger *zap.Logger, options ...MissingMessageVerifierOption) *MissingMessageVerifier {
	options = append(defaultMissingMessagesVerifierOptions, options...)
	params := missingMessageVerifierParams{}
	for _, opt := range options {
		opt(&params)
	}

	return &MissingMessageVerifier{
		storenodeRequestor: storenodeRequester,
		timesource:         timesource,
		messageTracker:     messageTracker,
		logger:             logger.Named("missing-msg-verifier"),
		params:             params,
		criteriaInterest:   make(map[string]*criteriaInterest),
		C:                  make(chan *protocol.Envelope, 1000),
	}
}

func (m *MissingMessageVerifier) SetCriteriaInterest(peerInfo peer.AddrInfo, contentFilter protocol.ContentFilter) {
	m.criteriaInterestMu.Lock()
	defer m.criteriaInterestMu.Unlock()

	ctx, cancel := context.WithCancel(m.ctx)
	criteriaInterest := criteriaInterest{
		peerInfo:      peerInfo,
		contentFilter: contentFilter,
		lastChecked:   m.timesource.Now().Add(-m.params.delay),
		ctx:           ctx,
		cancel:        cancel,
	}

	currMessageVerificationRequest, ok := m.criteriaInterest[contentFilter.PubsubTopic]

	if ok && currMessageVerificationRequest.equals(criteriaInterest) {
		return
	}

	if ok {
		// If there is an ongoing request, we cancel it before replacing it
		// by the new list. This can be probably optimized further by tracking
		// the last time a content topic was synced, but might not be necessary
		// since cancelling an ongoing request would mean cancelling just a single
		// page of results
		currMessageVerificationRequest.cancel()
	}

	m.criteriaInterest[contentFilter.PubsubTopic] = &criteriaInterest
}

func (m *MissingMessageVerifier) setRunning(running bool) {
	m.runningMutex.Lock()
	defer m.runningMutex.Unlock()
	m.isRunning = running
}

func (m *MissingMessageVerifier) Start(ctx context.Context) {
	m.runningMutex.Lock()
	if m.isRunning { //make sure verifier only runs once.
		m.runningMutex.Unlock()
		return
	}
	m.isRunning = true
	m.runningMutex.Unlock()

	ctx, cancelFunc := context.WithCancel(ctx)
	m.ctx = ctx
	m.cancel = cancelFunc

	// updating context for existing criteria
	m.criteriaInterestMu.Lock()
	for _, value := range m.criteriaInterest {
		ctx, cancel := context.WithCancel(m.ctx)
		value.ctx = ctx
		value.cancel = cancel
	}
	m.criteriaInterestMu.Unlock()

	go func() {
		defer utils.LogOnPanic()
		t := time.NewTicker(m.params.interval)
		defer t.Stop()

		var semaphore = make(chan struct{}, 5)
		for {
			select {
			case <-t.C:
				m.logger.Debug("checking for missing messages...")
				m.criteriaInterestMu.RLock()
				critIntList := make([]criteriaInterest, 0, len(m.criteriaInterest))
				for _, value := range m.criteriaInterest {
					critIntList = append(critIntList, *value)
				}
				m.criteriaInterestMu.RUnlock()
				for _, interest := range critIntList {
					select {
					case <-ctx.Done():
						m.setRunning(false)
						return
					default:
						semaphore <- struct{}{}
						go func(interest criteriaInterest) {
							defer utils.LogOnPanic()
							m.fetchHistory(m.C, interest)
							<-semaphore
						}(interest)
					}
				}

			case <-ctx.Done():
				m.setRunning(false)
				return
			}
		}
	}()
}

func (m *MissingMessageVerifier) Stop() {
	m.cancel()
	m.runningMutex.Lock()
	defer m.runningMutex.Unlock()
	m.isRunning = false
}

func (m *MissingMessageVerifier) fetchHistory(c chan<- *protocol.Envelope, interest criteriaInterest) {
	contentTopics := interest.contentFilter.ContentTopics.ToList()
	for i := 0; i < len(contentTopics); i += maxContentTopicsPerRequest {
		j := i + maxContentTopicsPerRequest
		if j > len(contentTopics) {
			j = len(contentTopics)
		}

		select {
		case <-interest.ctx.Done():
			return
		default:
			// continue...
		}

		now := m.timesource.Now()
		err := m.fetchMessagesBatch(c, interest, i, j, now)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}

			m.logger.Error("could not fetch history",
				zap.Stringer("peerID", interest.peerInfo.ID),
				zap.String("pubsubTopic", interest.contentFilter.PubsubTopic),
				zap.Strings("contentTopics", contentTopics))
			continue
		}

		m.criteriaInterestMu.Lock()
		c, ok := m.criteriaInterest[interest.contentFilter.PubsubTopic]
		if ok && c.equals(interest) {
			c.lastChecked = now
			m.criteriaInterest[interest.contentFilter.PubsubTopic] = c
		}
		m.criteriaInterestMu.Unlock()
	}
}

func (m *MissingMessageVerifier) storeQueryWithRetry(ctx context.Context, queryFunc func(ctx context.Context) (common.StoreRequestResult, error), logger *zap.Logger, logMsg string) (common.StoreRequestResult, error) {
	retry := true
	count := 1
	for retry && count <= m.params.maxAttemptsToRetrieveHistory {
		logger.Debug(logMsg, zap.Int("attempt", count))
		tCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
		result, err := queryFunc(tCtx)
		cancel()
		if err != nil {
			logger.Error("could not query storenode", zap.Error(err), zap.Int("attempt", count))
			select {
			case <-m.ctx.Done():
				return nil, m.ctx.Err()
			case <-time.After(2 * time.Second):
			}
		} else {
			return result, nil
		}
	}

	return nil, errors.New("storenode not available")
}

func (m *MissingMessageVerifier) fetchMessagesBatch(c chan<- *protocol.Envelope, interest criteriaInterest, batchFrom int, batchTo int, now time.Time) error {
	contentTopics := interest.contentFilter.ContentTopics.ToList()

	logger := m.logger.With(
		zap.Stringer("peerID", interest.peerInfo.ID),
		zap.Strings("contentTopics", contentTopics[batchFrom:batchTo]),
		zap.String("pubsubTopic", interest.contentFilter.PubsubTopic),
		logging.Epoch("from", interest.lastChecked),
		logging.Epoch("to", now),
	)

	result, err := m.storeQueryWithRetry(interest.ctx, func(ctx context.Context) (common.StoreRequestResult, error) {
		storeQueryRequest := &storepb.StoreQueryRequest{
			RequestId:       hex.EncodeToString(protocol.GenerateRequestID()),
			PubsubTopic:     &interest.contentFilter.PubsubTopic,
			ContentTopics:   contentTopics[batchFrom:batchTo],
			TimeStart:       proto.Int64(interest.lastChecked.Add(-m.params.delay).UnixNano()),
			TimeEnd:         proto.Int64(now.Add(-m.params.delay).UnixNano()),
			PaginationLimit: proto.Uint64(messageFetchPageSize),
		}

		return m.storenodeRequestor.Query(
			ctx,
			interest.peerInfo,
			storeQueryRequest,
		)
	}, logger, "retrieving history to check for missing messages")
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			logger.Error("storenode not available", zap.Error(err))
		}
		return err
	}

	var missingHashes []pb.MessageHash

	for !result.IsComplete() {
		for _, mkv := range result.Messages() {
			hash := pb.ToMessageHash(mkv.MessageHash)
			exists, err := m.messageTracker.MessageExists(hash)
			if err != nil {
				return err
			}

			if exists {
				continue
			}

			missingHashes = append(missingHashes, hash)
		}

		result, err = m.storeQueryWithRetry(interest.ctx, func(ctx context.Context) (common.StoreRequestResult, error) {
			if err = result.Next(ctx); err != nil {
				return nil, err
			}
			return result, nil
		}, logger.With(zap.String("cursor", hex.EncodeToString(result.Cursor()))), "retrieving next page")
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				logger.Error("storenode not available", zap.Error(err))
			}
			return err
		}
	}

	if len(missingHashes) == 0 {
		// Nothing to do here
		return nil
	}

	wg := sync.WaitGroup{}
	// Split into batches
	for i := 0; i < len(missingHashes); i += maxMsgHashesPerRequest {
		j := i + maxMsgHashesPerRequest
		if j > len(missingHashes) {
			j = len(missingHashes)
		}

		select {
		case <-interest.ctx.Done():
			return nil
		default:
			// continue...
		}

		wg.Add(1)
		go func(messageHashes []pb.MessageHash) {
			defer utils.LogOnPanic()
			defer wg.Wait()

			result, err := m.storeQueryWithRetry(interest.ctx, func(ctx context.Context) (common.StoreRequestResult, error) {
				queryCtx, cancel := context.WithTimeout(ctx, m.params.storeQueryTimeout)
				defer cancel()

				var messageHashesBytes [][]byte
				for _, m := range messageHashes {
					messageHashesBytes = append(messageHashesBytes, m.Bytes())
				}

				storeQueryRequest := &storepb.StoreQueryRequest{
					RequestId:       hex.EncodeToString(protocol.GenerateRequestID()),
					IncludeData:     true,
					MessageHashes:   messageHashesBytes,
					PaginationLimit: proto.Uint64(maxMsgHashesPerRequest),
				}

				return m.storenodeRequestor.Query(queryCtx, interest.peerInfo, storeQueryRequest)
			}, logger, "retrieving missing messages")
			if err != nil {
				if !errors.Is(err, context.Canceled) {
					logger.Error("storenode not available", zap.Error(err))
				}
				return
			}

			for !result.IsComplete() {
				for _, mkv := range result.Messages() {
					select {
					case c <- protocol.NewEnvelope(mkv.Message, mkv.Message.GetTimestamp(), mkv.GetPubsubTopic()):
					default:
						m.logger.Warn("subscriber is too slow!")
					}
				}

				result, err = m.storeQueryWithRetry(interest.ctx, func(ctx context.Context) (common.StoreRequestResult, error) {
					if err = result.Next(ctx); err != nil {
						return nil, err
					}
					return result, nil
				}, logger.With(zap.String("cursor", hex.EncodeToString(result.Cursor()))), "retrieving next page")
				if err != nil {
					if !errors.Is(err, context.Canceled) {
						logger.Error("storenode not available", zap.Error(err))
					}
					return
				}
			}

		}(missingHashes[i:j])
	}

	wg.Wait()

	return nil
}
