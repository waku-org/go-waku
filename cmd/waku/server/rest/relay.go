package rest

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/waku-org/go-waku/cmd/waku/server"
	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-waku/waku/v2/protocol"
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

	cache *MessageCache
}

// NewRelayService returns an instance of RelayService
func NewRelayService(node *node.WakuNode, m *chi.Mux, cacheCapacity int, log *zap.Logger) *RelayService {
	logger := log.Named("relay")

	s := &RelayService{
		node:  node,
		log:   logger,
		cache: NewMessageCache(cacheCapacity, logger),
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
		contentFilter := protocol.NewContentFilter(topic)
		err = r.node.Relay().Unsubscribe(req.Context(), contentFilter)
		if err != nil {
			r.log.Error("unsubscribing from topic", zap.String("topic", strings.Replace(strings.Replace(topic, "\n", "", -1), "\r", "", -1)), zap.Error(err))
		} else {
			err = r.cache.Unsubscribe(contentFilter)
			if err != nil {
				r.log.Error("unsubscribing from topic", zap.String("topic", strings.Replace(strings.Replace(topic, "\n", "", -1), "\r", "", -1)), zap.Error(err))
			}
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

		contentFilter := protocol.NewContentFilter(topicToSubscribe)
		subscriptions, err := r.node.Relay().Subscribe(r.node.Relay().Context(), contentFilter)
		if err != nil {
			r.log.Error("subscribing to topic", zap.String("topic", strings.Replace(topicToSubscribe, "\n", "", -1)), zap.Error(err))
			continue
		}

		err = r.cache.Subscribe(contentFilter)
		if err != nil {
			r.log.Error("subscribing cache to topic", zap.String("topic", strings.Replace(topicToSubscribe, "\n", "", -1)), zap.Error(err))
			continue
		}

		successCnt++

		for _, sub := range subscriptions {
			go func(sub *relay.Subscription) {
				for msg := range sub.Ch {
					r.cache.AddMessage(msg)
				}
			}(sub)
		}

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
	pubsubTopic := topicFromPath(w, req, "topic", r.log)
	if pubsubTopic == "" {
		return
	}

	//TODO: Update the API to also take a contentTopic since relay now supports filtering based on contentTopic as well.
	messages, err := r.cache.GetMessagesWithPubsubTopic(pubsubTopic)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_, err = w.Write([]byte("not subscribed to topic"))
		r.log.Error("writing response", zap.Error(err))
		return
	}

	writeErrOrResponse(w, nil, messages)
}

func (r *RelayService) postV1Message(w http.ResponseWriter, req *http.Request) {
	topic := topicFromPath(w, req, "topic", r.log)
	if topic == "" {
		return
	}

	var restMessage *RestWakuMessage
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&restMessage); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	if topic == "" {
		topic = relay.DefaultWakuTopic
	}

	message, err := restMessage.ToProto()
	if err != nil {
		writeErrOrResponse(w, err, nil)
		return
	}

	if err := server.AppendRLNProof(r.node, message); err != nil {
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

	contentFilter := protocol.NewContentFilter("", cTopics...)
	err := r.node.Relay().Unsubscribe(req.Context(), contentFilter)
	if err != nil {
		r.log.Error("unsubscribing from topics", zap.Strings("contentTopics", cTopics), zap.Error(err))
	} else {
		err = r.cache.Unsubscribe(contentFilter)
		if err != nil {
			r.log.Error("unsubscribing cache", zap.Strings("contentTopics", cTopics), zap.Error(err))
		}
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

	contentFilter := protocol.NewContentFilter("", cTopics...)
	subscriptions, err := r.node.Relay().Subscribe(r.node.Relay().Context(), contentFilter)
	if err != nil {
		r.log.Error("subscribing to topics", zap.Strings("contentTopics", cTopics), zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		_, err := w.Write([]byte(err.Error()))
		r.log.Error("writing response", zap.Error(err))
		return
	}

	err = r.cache.Subscribe(contentFilter)
	if err != nil {
		r.log.Error("subscribing cache failed", zap.Error(err))
	}

	for _, sub := range subscriptions {
		go func(sub *relay.Subscription) {
			for msg := range sub.Ch {
				r.cache.AddMessage(msg)
			}
		}(sub)
	}

	writeErrOrResponse(w, nil, true)

}

func (r *RelayService) getV1AutoMessages(w http.ResponseWriter, req *http.Request) {
	contentTopic := topicFromPath(w, req, "contentTopic", r.log)

	pubsubTopic, err := protocol.GetPubSubTopicFromContentTopic(contentTopic)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, err = w.Write([]byte("could not determine pubsubtopic"))
		r.log.Error("writing response", zap.Error(err))
		return
	}

	messages, err := r.cache.GetMessages(pubsubTopic, contentTopic)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_, err = w.Write([]byte("not subscribed to topic"))
		r.log.Error("writing response", zap.Error(err))
		return
	}

	writeErrOrResponse(w, nil, messages)
}

func (r *RelayService) postV1AutoMessage(w http.ResponseWriter, req *http.Request) {

	var restMessage *RestWakuMessage
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&restMessage); err != nil {
		r.log.Error("decoding message failure", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	message, err := restMessage.ToProto()
	if err != nil {
		writeErrOrResponse(w, err, nil)
		return
	}

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
