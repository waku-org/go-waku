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

// FilterService represents the REST service for Filter client
type FilterService struct {
	node *node.WakuNode

	log *zap.Logger
}

// NewFilterService returns an instance of FilterService
func NewFilterService(node *node.WakuNode, m *chi.Mux, log *zap.Logger) *FilterService {
	s := &FilterService{
		node: node,
		log:  log.Named("filter"),
	}

	m.Get(filterv2Ping, s.ping)
	m.Post(filterv2Subscribe, s.subscribe)
	m.Delete(filterv2Subscribe, s.unsubscribe)
	m.Delete(filterv2SubscribeAll, s.unsubscribeAll)

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

///////////////////////
///////////////////////

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

	//
	subscriptions, err := s.node.FilterLightnode().Subscribe(req.Context(),
		protocol.NewContentFilter(message.PubsubTopic, message.ContentFilters...),
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

	// unsubscribe on filter
	errCh, err := s.node.FilterLightnode().Unsubscribe(
		req.Context(),
		protocol.NewContentFilter(message.PubsubTopic, message.ContentFilters...),
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
	writeResponse(w, filterSubscriptionResponse{
		RequestId:  message.RequestId,
		StatusDesc: s.unsubscribeGetMessage(errCh),
	}, http.StatusOK)
}

func (s *FilterService) unsubscribeGetMessage(ch <-chan filter.WakuFilterPushResult) string {
	var peerIds string
	ind := 0
	for entry := range ch {
		if entry.Err == nil {
			continue
		}

		s.log.Error("can't unsubscribe for ", zap.Stringer("peer", entry.PeerID), zap.Error(entry.Err))
		if ind != 0 {
			peerIds += ", "
		}
		peerIds += entry.PeerID.String()
		ind++
	}
	if peerIds != "" {
		return "can't unsubscribe from " + peerIds
	}
	return http.StatusText(http.StatusOK)
}

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
