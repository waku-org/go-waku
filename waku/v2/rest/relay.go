package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"go.uber.org/zap"
)

const ROUTE_RELAY_SUBSCRIPTIONSV1 = "/relay/v1/subscriptions"
const ROUTE_RELAY_MESSAGESV1 = "/relay/v1/messages/{topic}"

type RelayService struct {
	node   *node.WakuNode
	cancel context.CancelFunc

	log *zap.Logger

	messages      map[string][]*pb.WakuMessage
	cacheCapacity int
	messagesMutex sync.RWMutex

	runner *runnerService
}

func NewRelayService(node *node.WakuNode, m *chi.Mux, cacheCapacity int, log *zap.Logger) *RelayService {
	s := &RelayService{
		node:          node,
		log:           log.Named("relay"),
		cacheCapacity: cacheCapacity,
		messages:      make(map[string][]*pb.WakuMessage),
	}

	s.runner = newRunnerService(node.Broadcaster(), s.addEnvelope)

	m.Post(ROUTE_RELAY_SUBSCRIPTIONSV1, s.postV1Subscriptions)
	m.Delete(ROUTE_RELAY_SUBSCRIPTIONSV1, s.deleteV1Subscriptions)
	m.Get(ROUTE_RELAY_MESSAGESV1, s.getV1Messages)
	m.Post(ROUTE_RELAY_MESSAGESV1, s.postV1Message)

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

func (r *RelayService) Stop() {
	if r.cancel == nil {
		return
	}
	r.cancel()
}

func (d *RelayService) deleteV1Subscriptions(w http.ResponseWriter, r *http.Request) {
	var topics []string
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&topics); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	d.messagesMutex.Lock()
	defer d.messagesMutex.Unlock()

	var err error
	for _, topic := range topics {
		err = d.node.Relay().Unsubscribe(r.Context(), topic)
		if err != nil {
			d.log.Error("unsubscribing from topic", zap.String("topic", strings.Replace(strings.Replace(topic, "\n", "", -1), "\r", "", -1)), zap.Error(err))
		} else {
			delete(d.messages, topic)
		}
	}

	writeErrOrResponse(w, err, true)
}

func (d *RelayService) postV1Subscriptions(w http.ResponseWriter, r *http.Request) {
	var topics []string
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&topics); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var err error
	var sub *relay.Subscription
	var topicToSubscribe string
	for _, topic := range topics {
		if topic == "" {
			sub, err = d.node.Relay().Subscribe(r.Context())
			topicToSubscribe = relay.DefaultWakuTopic
		} else {
			sub, err = d.node.Relay().SubscribeToTopic(r.Context(), topic)
			topicToSubscribe = topic
		}
		if err != nil {
			d.log.Error("subscribing to topic", zap.String("topic", strings.Replace(topicToSubscribe, "\n", "", -1)), zap.Error(err))
		} else {
			d.node.Broadcaster().Unregister(&topicToSubscribe, sub.C)
			d.messagesMutex.Lock()
			d.messages[topic] = []*pb.WakuMessage{}
			d.messagesMutex.Unlock()
		}
	}

	writeErrOrResponse(w, err, true)
}

func (d *RelayService) getV1Messages(w http.ResponseWriter, r *http.Request) {
	topic := chi.URLParam(r, "topic")
	if topic == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var err error

	d.messagesMutex.Lock()
	defer d.messagesMutex.Unlock()

	if _, ok := d.messages[topic]; !ok {
		w.WriteHeader(http.StatusNotFound)
		_, err = w.Write([]byte("not subscribed to topic"))
		d.log.Error("writing response", zap.Error(err))
		return
	}

	response := d.messages[topic]

	d.messages[topic] = []*pb.WakuMessage{}
	writeErrOrResponse(w, nil, response)
}

func (d *RelayService) postV1Message(w http.ResponseWriter, r *http.Request) {
	topic := chi.URLParam(r, "topic")
	if topic == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var message *pb.WakuMessage
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&message); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var err error
	if topic == "" {
		_, err = d.node.Relay().Publish(r.Context(), message)
	} else {
		_, err = d.node.Relay().PublishToTopic(r.Context(), message, strings.Replace(topic, "\n", "", -1))
	}

	if err != nil {
		d.log.Error("publishing message", zap.Error(err))
	}

	writeErrOrResponse(w, err, true)
}
