package rest

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/waku-org/go-waku/logging"
	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-waku/waku/v2/peermanager"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/filter"
	"go.uber.org/zap"
)

type filterRequestId []byte

func (r *filterRequestId) UnmarshalJSON(bodyBytes []byte) error {
	body := strings.Trim(string(bodyBytes), `"`)
	reqId, err := hex.DecodeString(body)
	if err != nil {
		return err
	}
	*r = reqId
	return nil
}

func (r filterRequestId) String() string {
	return hex.EncodeToString(r)
}
func (r filterRequestId) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, r.String())), nil
}

const filterv2Ping = "/filter/v2/subscriptions/{requestId}"
const filterv2Subscribe = "/filter/v2/subscriptions"
const filterv2SubscribeAll = "/filter/v2/subscriptions/all"
const filterv2MessagesByContentTopic = "/filter/v2/messages/{contentTopic}"
const filterv2MessagesByPubsubTopic = "/filter/v2/messages/{pubsubTopic}/{contentTopic}"

// FilterService represents the REST service for Filter client
type FilterService struct {
	node   *node.WakuNode
	cancel context.CancelFunc

	log *zap.Logger

	cache  *filterCache
	runner *runnerService
}

// Start starts the RelayService
func (s *FilterService) Start(ctx context.Context) {

	for _, sub := range s.node.FilterLightnode().Subscriptions() {
		s.cache.subscribe(sub.ContentFilter)
	}

	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.runner.Start(ctx)
}

// Stop stops the RelayService
func (r *FilterService) Stop() {
	if r.cancel == nil {
		return
	}
	r.cancel()
}

// NewFilterService returns an instance of FilterService
func NewFilterService(node *node.WakuNode, m *chi.Mux, cacheCapacity int, log *zap.Logger) *FilterService {
	logger := log.Named("filter")

	s := &FilterService{
		node:  node,
		log:   logger,
		cache: newFilterCache(cacheCapacity, logger),
	}

	m.Get(filterv2Ping, s.ping)
	m.Post(filterv2Subscribe, s.subscribe)
	m.Delete(filterv2Subscribe, s.unsubscribe)
	m.Delete(filterv2SubscribeAll, s.unsubscribeAll)
	m.Get(filterv2MessagesByContentTopic, s.getMessagesByContentTopic)
	m.Get(filterv2MessagesByPubsubTopic, s.getMessagesByPubsubTopic)

	s.runner = newRunnerService(node.Broadcaster(), s.cache.addMessage)

	return s
}

// 400 for bad requestId
// 404 when request failed or no suitable peers
// 200 when ping successful
func (s *FilterService) ping(w http.ResponseWriter, req *http.Request) {
	var requestId filterRequestId
	if err := requestId.UnmarshalJSON([]byte(chi.URLParam(req, "requestId"))); err != nil {
		s.log.Error("bad request id", zap.Error(err))
		writeResponse(w, &filterSubscriptionResponse{
			RequestId:  requestId,
			StatusDesc: "bad request id",
		}, http.StatusBadRequest)
		return
	}

	// selecting random peer that supports filter protocol
	peerId := s.getRandomFilterPeer(req.Context(), requestId, w)
	if peerId == "" {
		return
	}

	if err := s.node.FilterLightnode().Ping(req.Context(), peerId, filter.WithPingRequestId(requestId)); err != nil {
		s.log.Error("ping request failed", zap.Error(err))
		writeResponse(w, &filterSubscriptionResponse{
			RequestId:  requestId,
			StatusDesc: "ping request failed",
		}, http.StatusServiceUnavailable)
		return
	}

	// success
	writeResponse(w, &filterSubscriptionResponse{
		RequestId:  requestId,
		StatusDesc: http.StatusText(http.StatusOK),
	}, http.StatusOK)
}

// same for FilterUnsubscribeRequest
type filterSubscriptionRequest struct {
	RequestId      filterRequestId `json:"requestId"`
	ContentFilters []string        `json:"contentFilters"`
	PubsubTopic    string          `json:"pubsubTopic"`
}

type filterSubscriptionResponse struct {
	RequestId  filterRequestId `json:"requestId"`
	StatusDesc string          `json:"statusDesc"`
}

// 400 on invalid request
// 404 on failed subscription
// 200 on single returned successful subscription
// NOTE: subscribe on filter client randomly selects a peer if missing for given pubSubTopic
func (s *FilterService) subscribe(w http.ResponseWriter, req *http.Request) {
	message := filterSubscriptionRequest{}
	if !s.readBody(w, req, &message) {
		return
	}

	contentFilter := protocol.NewContentFilter(message.PubsubTopic, message.ContentFilters...)
	//
	subscriptions, err := s.node.FilterLightnode().Subscribe(req.Context(),
		contentFilter,
		filter.WithRequestID(message.RequestId))

	// on partial subscribe failure
	if len(subscriptions) > 0 && err != nil {
		s.log.Error("partial subscribe failed", zap.Error(err))
		// on partial failure
		writeResponse(w, filterSubscriptionResponse{
			RequestId:  message.RequestId,
			StatusDesc: err.Error(),
		}, http.StatusOK)
	}

	if err != nil {
		s.log.Error("subscription failed", zap.Error(err))
		writeResponse(w, filterSubscriptionResponse{
			RequestId:  message.RequestId,
			StatusDesc: "subscription failed",
		}, http.StatusServiceUnavailable)
		return
	}

	// on success
	s.cache.subscribe(contentFilter)
	writeResponse(w, filterSubscriptionResponse{
		RequestId:  message.RequestId,
		StatusDesc: http.StatusText(http.StatusOK),
	}, http.StatusOK)
}

// 400 on invalid request
// 500 on failed subscription
// 200 on successful unsubscribe
// NOTE: unsubscribe on filter client will remove subscription from all peers with matching pubSubTopic, if peerId is not provided
// to match functionality in nwaku, we will randomly select a peer that supports filter protocol.
func (s *FilterService) unsubscribe(w http.ResponseWriter, req *http.Request) {
	message := filterSubscriptionRequest{} // as pubSubTopics can also be present
	if !s.readBody(w, req, &message) {
		return
	}

	peerId := s.getRandomFilterPeer(req.Context(), message.RequestId, w)
	if peerId == "" {
		return
	}

	contentFilter := protocol.NewContentFilter(message.PubsubTopic, message.ContentFilters...)
	// unsubscribe on filter
	result, err := s.node.FilterLightnode().Unsubscribe(
		req.Context(),
		contentFilter,
		filter.WithRequestID(message.RequestId),
		filter.WithPeer(peerId),
	)

	if err != nil {
		s.log.Error("unsubscribe failed", zap.Error(err))
		writeResponse(w, filterSubscriptionResponse{
			RequestId:  message.RequestId,
			StatusDesc: err.Error(),
		}, http.StatusServiceUnavailable)
		return
	}

	// on success
	for cTopic := range contentFilter.ContentTopics {
		if !s.node.FilterLightnode().IsListening(contentFilter.PubsubTopic, cTopic) {
			s.cache.unsubscribe(contentFilter.PubsubTopic, cTopic)
		}
	}
	writeResponse(w, filterSubscriptionResponse{
		RequestId:  message.RequestId,
		StatusDesc: s.unsubscribeGetMessage(result),
	}, http.StatusOK)
}

func (s *FilterService) unsubscribeGetMessage(result *filter.WakuFilterPushResult) string {
	if result == nil {
		return http.StatusText(http.StatusOK)
	}
	var peerIds string
	ind := 0
	for _, entry := range result.Errors() {
		if entry.Err != nil {
			s.log.Error("can't unsubscribe", logging.HostID("peer", entry.PeerID), zap.Error(entry.Err))
			if ind != 0 {
				peerIds += ", "
			}
			peerIds += entry.PeerID.String()
		}
		ind++
	}
	if peerIds != "" {
		return "can't unsubscribe from " + peerIds
	}
	return http.StatusText(http.StatusOK)
}

// ///////////////////////
// ///////////////////////
type filterUnsubscribeAllRequest struct {
	RequestId filterRequestId `json:"requestId"`
}

func (s *FilterService) readBody(w http.ResponseWriter, req *http.Request, message interface{}) bool {
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(message); err != nil {
		s.log.Error("bad request", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		return false
	}
	defer req.Body.Close()
	return true
}

// 400 on invalid request
// 500 on failed subscription
// 200 on all successful unsubscribe
// unsubscribe all subscriptions for a given peer
func (s *FilterService) unsubscribeAll(w http.ResponseWriter, req *http.Request) {
	message := filterUnsubscribeAllRequest{}
	if !s.readBody(w, req, &message) {
		return
	}

	peerId := s.getRandomFilterPeer(req.Context(), message.RequestId, w)
	if peerId == "" {
		return
	}

	// unsubscribe all subscriptions for a given peer
	errCh, err := s.node.FilterLightnode().UnsubscribeAll(
		req.Context(),
		filter.WithRequestID(message.RequestId),
		filter.WithPeer(peerId),
	)
	if err != nil {
		s.log.Error("unsubscribeAll failed", zap.Error(err))
		writeResponse(w, filterSubscriptionResponse{
			RequestId:  message.RequestId,
			StatusDesc: err.Error(),
		}, http.StatusServiceUnavailable)
		return
	}

	// on success
	writeResponse(w, filterSubscriptionResponse{
		RequestId:  message.RequestId,
		StatusDesc: s.unsubscribeGetMessage(errCh),
	}, http.StatusOK)
}

func (s FilterService) getRandomFilterPeer(ctx context.Context, requestId []byte, w http.ResponseWriter) peer.ID {
	// selecting random peer that supports filter protocol
	peerId, err := s.node.PeerManager().SelectPeer(peermanager.PeerSelectionCriteria{
		SelectionType: peermanager.Automatic,
		Proto:         filter.FilterSubscribeID_v20beta1,
		Ctx:           ctx,
	})
	if err != nil {
		s.log.Error("selecting peer", zap.Error(err))
		writeResponse(w, filterSubscriptionResponse{
			RequestId:  requestId,
			StatusDesc: "No suitable peers",
		}, http.StatusServiceUnavailable)
		return ""
	}
	return peerId
}

func (s *FilterService) getMessagesByContentTopic(w http.ResponseWriter, req *http.Request) {
	contentTopic := topicFromPath(w, req, "contentTopic", s.log)
	if contentTopic == "" {
		return
	}
	pubsubTopic, err := protocol.GetPubSubTopicFromContentTopic(contentTopic)
	if err != nil {
		writeGetMessageErr(w, fmt.Errorf("bad content topic"), http.StatusBadRequest, s.log)
		return
	}
	s.getMessages(w, req, pubsubTopic, contentTopic)
}

func (s *FilterService) getMessagesByPubsubTopic(w http.ResponseWriter, req *http.Request) {
	contentTopic := topicFromPath(w, req, "contentTopic", s.log)
	if contentTopic == "" {
		return
	}
	pubsubTopic := topicFromPath(w, req, "pubsubTopic", s.log)
	if pubsubTopic == "" {
		return
	}
	s.getMessages(w, req, pubsubTopic, contentTopic)
}

// 400 on invalid request
// 500 on failed subscription
// 200 on all successful unsubscribe
// unsubscribe all subscriptions for a given peer
func (s *FilterService) getMessages(w http.ResponseWriter, req *http.Request, pubsubTopic, contentTopic string) {
	msgs, err := s.cache.getMessages(pubsubTopic, contentTopic)
	if err != nil {
		writeGetMessageErr(w, err, http.StatusNotFound, s.log)
		return
	}
	writeResponse(w, msgs, http.StatusOK)
}
