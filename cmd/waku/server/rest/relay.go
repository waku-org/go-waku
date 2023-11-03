package rest

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"sync"

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
	node   *node.WakuNode
	cancel context.CancelFunc

	log *zap.Logger

	messages      map[string][]*pb.WakuMessage
	cacheCapacity int
	messagesMutex sync.RWMutex

	runner *runnerService
}

// NewRelayService returns an instance of RelayService
func NewRelayService(node *node.WakuNode, m *chi.Mux, cacheCapacity int, log *zap.Logger) *RelayService {
	s := &RelayService{
		node:          node,
		log:           log.Named("relay"),
		cacheCapacity: cacheCapacity,
		messages:      make(map[string][]*pb.WakuMessage),
	}

	s.runner = newRunnerService(node.Broadcaster(), s.addEnvelope)

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

func (r *RelayService) addEnvelope(envelope *protocol.Envelope) {
	r.messagesMutex.Lock()
	defer r.messagesMutex.Unlock()

	if _, ok := r.messages[envelope.PubsubTopic()]; !ok {
		return
	}

	// Keep a specific max number of messages per topic
	if len(r.messages[envelope.PubsubTopic()]) >= r.cacheCapacity {
		r.messages[envelope.PubsubTopic()] = r.messages[envelope.PubsubTopic()][1:]
	}

	r.messages[envelope.PubsubTopic()] = append(r.messages[envelope.PubsubTopic()], envelope.Message())
}

// Start starts the RelayService
func (r *RelayService) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	r.cancel = cancel

	r.messagesMutex.Lock()
	// Node may already be subscribed to some topics when Relay API handlers are installed. Let's add these
	for _, topic := range r.node.Relay().Topics() {
		r.log.Info("adding topic handler for existing subscription", zap.String("topic", topic))
		r.messages[topic] = []*pb.WakuMessage{}
	}
	r.messagesMutex.Unlock()

	r.runner.Start(ctx)
}

// Stop stops the RelayService
func (r *RelayService) Stop() {
	if r.cancel == nil {
		return
	}
	r.cancel()
}

func (r *RelayService) deleteV1Subscriptions(w http.ResponseWriter, req *http.Request) {
	var topics []string
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&topics); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	r.messagesMutex.Lock()
	defer r.messagesMutex.Unlock()

	var err error
	for _, topic := range topics {
		err = r.node.Relay().Unsubscribe(req.Context(), protocol.NewContentFilter(topic))
		if err != nil {
			r.log.Error("unsubscribing from topic", zap.String("topic", strings.Replace(strings.Replace(topic, "\n", "", -1), "\r", "", -1)), zap.Error(err))
		} else {
			delete(r.messages, topic)
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
	var sub *relay.Subscription
	var subs []*relay.Subscription
	var topicToSubscribe string
	for _, topic := range topics {
		if topic == "" {
			subs, err = r.node.Relay().Subscribe(req.Context(), protocol.NewContentFilter(relay.DefaultWakuTopic))
			topicToSubscribe = relay.DefaultWakuTopic
		} else {
			subs, err = r.node.Relay().Subscribe(req.Context(), protocol.NewContentFilter(topic))
			topicToSubscribe = topic
		}
		if err != nil {
			r.log.Error("subscribing to topic", zap.String("topic", strings.Replace(topicToSubscribe, "\n", "", -1)), zap.Error(err))
		} else {
			sub = subs[0]
			sub.Unsubscribe()
			r.messagesMutex.Lock()
			r.messages[topic] = []*pb.WakuMessage{}
			r.messagesMutex.Unlock()
		}
	}

	writeErrOrResponse(w, err, true)
}

func (r *RelayService) getV1Messages(w http.ResponseWriter, req *http.Request) {
	topic := chi.URLParam(req, "topic")
	if topic == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	topic, err := url.QueryUnescape(topic)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, err = w.Write([]byte("invalid topic format"))
		r.log.Error("writing response", zap.Error(err))
		return
	}

	r.messagesMutex.Lock()
	defer r.messagesMutex.Unlock()

	if _, ok := r.messages[topic]; !ok {
		w.WriteHeader(http.StatusNotFound)
		_, err = w.Write([]byte("not subscribed to topic"))
		r.log.Error("writing response", zap.Error(err))
		return
	}

	response := r.messages[topic]

	r.messages[topic] = []*pb.WakuMessage{}
	writeErrOrResponse(w, nil, response)
}

func (r *RelayService) postV1Message(w http.ResponseWriter, req *http.Request) {
	topic := chi.URLParam(req, "topic")
	if topic == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	topic, err := url.QueryUnescape(topic)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, err = w.Write([]byte("invalid topic format"))
		r.log.Error("writing response", zap.Error(err))
		return
	}
	var message *pb.WakuMessage
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&message); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	//var err error
	if topic == "" {
		topic = relay.DefaultWakuTopic
	}

	if !r.node.Relay().IsSubscribed(topic) {
		writeErrOrResponse(w, errors.New("not subscribed to pubsubTopic"), nil)
		return
	}

	if err = server.AppendRLNProof(r.node, message); err != nil {
		writeErrOrResponse(w, err, nil)
		return
	}

	_, err = r.node.Relay().Publish(req.Context(), message, relay.WithPubSubTopic(strings.Replace(topic, "\n", "", -1)))
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
	_, err = r.node.Relay().Subscribe(r.node.Relay().Context(), protocol.NewContentFilter("", cTopics...))
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
	cTopic := chi.URLParam(req, "contentTopic")
	if cTopic == "" {
		w.WriteHeader(http.StatusBadRequest)
		_, err := w.Write([]byte("contentTopic is required"))
		r.log.Error("writing response", zap.Error(err))
		return
	}
	cTopic, err := url.QueryUnescape(cTopic)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, err = w.Write([]byte("invalid contentTopic format"))
		r.log.Error("writing response", zap.Error(err))
		return
	}

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
