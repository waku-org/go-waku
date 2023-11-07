package rest

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/waku-org/go-waku/cmd/waku/server"
	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"go.uber.org/zap"
)

const routeRelayV1Subscriptions = "/relay/v1/subscriptions"
const routeRelayV1Messages = "/relay/v1/messages/{topic}"

const routeRelayV1AutoSubscriptions = "/relay/v1/auto/subscriptions"
const routeRelayV1AutoMessages = "/relay/v1/auto/messages"

// RelayService represents the REST service for WakuRelay
type RelayService struct {
	node *node.WakuNode

	log *zap.Logger

	cacheCapacity uint
}

// NewRelayService returns an instance of RelayService
func NewRelayService(node *node.WakuNode, m *chi.Mux, cacheCapacity uint, log *zap.Logger) *RelayService {
	s := &RelayService{
		node:          node,
		log:           log.Named("relay"),
		cacheCapacity: cacheCapacity,
	}

	m.Post(routeRelayV1Subscriptions, s.postV1Subscriptions)
	m.Delete(routeRelayV1Subscriptions, s.deleteV1Subscriptions)
	m.Get(routeRelayV1Messages, s.getV1Messages)
	m.Post(routeRelayV1Messages, s.postV1Message)

	m.Post(routeRelayV1AutoSubscriptions, s.postV1AutoSubscriptions)
	m.Delete(routeRelayV1AutoSubscriptions, s.deleteV1AutoSubscriptions)

	m.Route(routeRelayV1AutoMessages, func(r chi.Router) {
		r.Get("/{contentTopic}", s.getV1AutoMessages)
		r.Post("/", s.postV1AutoMessage)
	})

	return s
}

func (r *RelayService) deleteV1Subscriptions(w http.ResponseWriter, req *http.Request) {
	var topics []string
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&topics); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	var err error
	for _, topic := range topics {
		err = r.node.Relay().Unsubscribe(req.Context(), protocol.NewContentFilter(topic))
		if err != nil {
			r.log.Error("unsubscribing from topic", zap.String("topic", strings.Replace(strings.Replace(topic, "\n", "", -1), "\r", "", -1)), zap.Error(err))
		}
	}

	writeErrOrResponse(w, err, true)
}

func (r *RelayService) postV1Subscriptions(w http.ResponseWriter, req *http.Request) {
	var topics []string
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&topics); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	var err error
	var successCnt int
	var topicToSubscribe string
	for _, topic := range topics {
		if topic == "" {
			topicToSubscribe = relay.DefaultWakuTopic
		} else {
			topicToSubscribe = topic
		}
		_, err = r.node.Relay().Subscribe(req.Context(), protocol.NewContentFilter(topicToSubscribe), relay.WithCacheSize(r.cacheCapacity))

		if err != nil {
			r.log.Error("subscribing to topic", zap.String("topic", strings.Replace(topicToSubscribe, "\n", "", -1)), zap.Error(err))
			continue
		}
		successCnt++
	}

	// on partial subscribe failure
	if successCnt > 0 && err != nil {
		r.log.Error("partial subscribe failed", zap.Error(err))
		// on partial failure
		writeResponse(w, err, http.StatusOK)
		return
	}

	writeErrOrResponse(w, err, true)
}

func (r *RelayService) getV1Messages(w http.ResponseWriter, req *http.Request) {
	topic := topicFromPath(w, req, "topic", r.log)
	if topic == "" {
		return
	}
	//TODO: Update the API to also take a contentTopic since relay now supports filtering based on contentTopic as well.
	sub, err := r.node.Relay().GetSubscriptionWithPubsubTopic(topic, "")
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_, err = w.Write([]byte("not subscribed to topic"))
		r.log.Error("writing response", zap.Error(err))
		return
	}
	var response []*pb.WakuMessage
	select {
	case msg := <-sub.Ch:
		response = append(response, msg.Message())
	default:
		break
	}

	writeErrOrResponse(w, nil, response)
}

func (r *RelayService) postV1Message(w http.ResponseWriter, req *http.Request) {
	topic := topicFromPath(w, req, "topic", r.log)
	if topic == "" {
		return
	}

	var message *pb.WakuMessage
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&message); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	if topic == "" {
		topic = relay.DefaultWakuTopic
	}

	if err := server.AppendRLNProof(r.node, message); err != nil {
		writeErrOrResponse(w, err, nil)
		return
	}

	_, err := r.node.Relay().Publish(req.Context(), message, relay.WithPubSubTopic(strings.Replace(topic, "\n", "", -1)))
	if err != nil {
		r.log.Error("publishing message", zap.Error(err))
	}

	writeErrOrResponse(w, err, true)
}

func (r *RelayService) deleteV1AutoSubscriptions(w http.ResponseWriter, req *http.Request) {
	var cTopics []string
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&cTopics); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	err := r.node.Relay().Unsubscribe(req.Context(), protocol.NewContentFilter("", cTopics...))
	if err != nil {
		r.log.Error("unsubscribing from topics", zap.Strings("contentTopics", cTopics), zap.Error(err))
	}

	writeErrOrResponse(w, err, true)
}

func (r *RelayService) postV1AutoSubscriptions(w http.ResponseWriter, req *http.Request) {
	var cTopics []string
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&cTopics); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	var err error
	_, err = r.node.Relay().Subscribe(r.node.Relay().Context(), protocol.NewContentFilter("", cTopics...), relay.WithCacheSize(r.cacheCapacity))
	if err != nil {
		r.log.Error("subscribing to topics", zap.Strings("contentTopics", cTopics), zap.Error(err))
	}

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, err := w.Write([]byte(err.Error()))
		r.log.Error("writing response", zap.Error(err))
	} else {
		w.WriteHeader(http.StatusOK)
	}

}

func (r *RelayService) getV1AutoMessages(w http.ResponseWriter, req *http.Request) {

	cTopic := topicFromPath(w, req, "contentTopic", r.log)
	sub, err := r.node.Relay().GetSubscription(cTopic)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_, err = w.Write([]byte("not subscribed to topic"))
		r.log.Error("writing response", zap.Error(err))
		return
	}
	var response []*pb.WakuMessage
	select {
	case msg := <-sub.Ch:
		response = append(response, msg.Message())
	default:
		break
	}

	writeErrOrResponse(w, nil, response)
}

func (r *RelayService) postV1AutoMessage(w http.ResponseWriter, req *http.Request) {

	var message *pb.WakuMessage
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&message); err != nil {
		r.log.Error("decoding message failure", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer req.Body.Close()
	var err error
	if err = server.AppendRLNProof(r.node, message); err != nil {
		writeErrOrResponse(w, err, nil)
		return
	}

	_, err = r.node.Relay().Publish(req.Context(), message)
	if err != nil {
		r.log.Error("publishing message", zap.Error(err))
	}

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, err := w.Write([]byte(err.Error()))
		r.log.Error("writing response", zap.Error(err))
	} else {
		w.WriteHeader(http.StatusOK)
	}

}
