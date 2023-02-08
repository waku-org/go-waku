package filterv2

import (
	"context"
	"encoding/hex"
	"errors"
	"math"
	"sync"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	libp2pProtocol "github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-msgio/protoio"
	"github.com/waku-org/go-waku/logging"
	v2 "github.com/waku-org/go-waku/waku/v2"
	"github.com/waku-org/go-waku/waku/v2/metrics"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/timesource"
	"go.opencensus.io/tag"
	"go.uber.org/zap"
)

// FilterPushID_v20beta1 is the current Waku Filter protocol identifier used to allow
// filter service nodes to push messages matching registered subscriptions to this client.
const FilterPushID_v20beta1 = libp2pProtocol.ID("/vac/waku/filter-push/2.0.0-beta1")

var (
	ErrNoPeersAvailable = errors.New("no suitable remote peers")
)

type WakuFilterPush struct {
	cancel        context.CancelFunc
	ctx           context.Context
	h             host.Host
	broadcaster   v2.Broadcaster
	timesource    timesource.Timesource
	wg            *sync.WaitGroup
	log           *zap.Logger
	subscriptions *SubscriptionsMap
}

type ContentFilter struct {
	Topic         string
	ContentTopics []string
}

// NewWakuRelay returns a new instance of Waku Filter struct setup according to the chosen parameter and options
func NewWakuFilterPush(host host.Host, broadcaster v2.Broadcaster, timesource timesource.Timesource, log *zap.Logger) *WakuFilterPush {
	wf := new(WakuFilterPush)
	wf.log = log.Named("filter")
	wf.broadcaster = broadcaster
	wf.timesource = timesource
	wf.wg = &sync.WaitGroup{}
	wf.h = host

	return wf
}

func (wf *WakuFilterPush) Start(ctx context.Context) error {
	wf.wg.Wait() // Wait for any goroutines to stop

	ctx, err := tag.New(ctx, tag.Insert(metrics.KeyType, "filter"))
	if err != nil {
		wf.log.Error("creating tag map", zap.Error(err))
		return errors.New("could not start waku filter")
	}

	ctx, cancel := context.WithCancel(ctx)
	wf.cancel = cancel
	wf.ctx = ctx
	wf.subscriptions = NewSubscriptionMap()

	wf.h.SetStreamHandlerMatch(FilterPushID_v20beta1, protocol.PrefixTextMatch(string(FilterPushID_v20beta1)), wf.onRequest(ctx))

	wf.wg.Add(1)

	// TODO: go wf.keepAliveSubscriptions(ctx)

	wf.log.Info("filter protocol (light) started")

	return nil
}

// Stop unmounts the filter protocol
func (wf *WakuFilterPush) Stop() {
	if wf.cancel == nil {
		return
	}

	wf.cancel()

	wf.h.RemoveStreamHandler(FilterPushID_v20beta1)

	wf.UnsubscribeAll(wf.ctx)

	wf.subscriptions.Clear()

	wf.wg.Wait()
}

func (wf *WakuFilterPush) onRequest(ctx context.Context) func(s network.Stream) {
	return func(s network.Stream) {
		defer s.Close()
		logger := wf.log.With(logging.HostID("peer", s.Conn().RemotePeer()))

		reader := protoio.NewDelimitedReader(s, math.MaxInt32)

		messagePush := &pb.MessagePushV2{}
		err := reader.ReadMsg(messagePush)
		if err != nil {
			logger.Error("reading message push", zap.Error(err))
			return
		}

		wf.notify(s.Conn().RemotePeer(), messagePush.PubsubTopic, messagePush.WakuMessage)

		logger.Info("received message push")
	}
}

func (wf *WakuFilterPush) notify(remotePeerID peer.ID, pubsubTopic string, msg *pb.WakuMessage) {
	envelope := protocol.NewEnvelope(msg, wf.timesource.Now().UnixNano(), pubsubTopic)

	// Broadcasting message so it's stored
	wf.broadcaster.Submit(envelope)

	// Notify filter subscribers
	wf.subscriptions.Notify(remotePeerID, envelope)
}

func (wf *WakuFilterPush) request(ctx context.Context, params *FilterSubscribeParameters, reqType pb.FilterSubscribeRequest_FilterSubscribeType, contentFilter ContentFilter) error {
	err := wf.h.Connect(ctx, wf.h.Peerstore().PeerInfo(params.selectedPeer))
	if err != nil {
		return err
	}

	var conn network.Stream
	conn, err = wf.h.NewStream(ctx, params.selectedPeer, FilterSubscribeID_v20beta1)
	if err != nil {
		return err
	}

	writer := protoio.NewDelimitedWriter(conn)

	request := &pb.FilterSubscribeRequest{
		RequestId:           hex.EncodeToString(params.requestId),
		FilterSubscribeType: reqType,
		PubsubTopic:         contentFilter.Topic,
		ContentTopics:       contentFilter.ContentTopics,
	}

	wf.log.Debug("sending FilterSubscribeRequest", zap.Stringer("request", request))
	err = writer.WriteMsg(request)
	if err != nil {
		wf.log.Error("sending FilterSubscribeRequest", zap.Error(err))
		return err
	}

	defer conn.Close()
	return nil
}

// Subscribe setups a subscription to receive messages that match a specific content filter
func (wf *WakuFilterPush) Subscribe(ctx context.Context, contentFilter ContentFilter, opts ...FilterSubscribeOption) error {
	// TODO: validate content filters

	params := new(FilterSubscribeParameters)
	params.log = wf.log
	params.host = wf.h

	optList := DefaultSubscriptionOptions()
	optList = append(optList, opts...)
	for _, opt := range optList {
		opt(params)
	}

	if params.selectedPeer == "" {
		return ErrNoPeersAvailable
	}

	err := wf.request(ctx, params, pb.FilterSubscribeRequest_SUBSCRIBE, contentFilter)
	if err != nil {
		return err
	}

	return nil
}

// SubscriptionChannel is used to obtain an object from which you could receive messages received via filter protocol
func (wf *WakuFilterPush) SubscriptionChannel(peerID peer.ID, topic string, contentTopics []string) *SubscriptionDetails {
	return wf.subscriptions.NewSubscription(peerID, topic, contentTopics)
}

func (wf *WakuFilterPush) getUnsubscribeParameters(opts ...FilterUnsubscribeOption) (*FilterUnsubscribeParameters, error) {
	params := new(FilterUnsubscribeParameters)
	params.log = wf.log
	for _, opt := range opts {
		opt(params)
	}

	if !params.unsubscribeAll && params.selectedPeer == "" {
		return nil, ErrNoPeersAvailable
	}

	return params, nil
}

// Unsubscribe is used to stop receiving messages from a peer that match a content filter
func (wf *WakuFilterPush) Unsubscribe(ctx context.Context, contentFilter ContentFilter, opts ...FilterUnsubscribeOption) error {
	// TODO: checks if a subscription exists with the chosen criteria

	params, err := wf.getUnsubscribeParameters(opts...)
	if err != nil {
		return err
	}

	for peerID := range wf.subscriptions.items {
		if !params.unsubscribeAll && peerID != params.selectedPeer {
			continue
		}

		go func(peerID peer.ID) {
			defer wf.wg.Done()
			err := wf.request(
				ctx,
				&FilterSubscribeParameters{selectedPeer: peerID},
				pb.FilterSubscribeRequest_UNSUBSCRIBE,
				ContentFilter{})
			if err != nil {
				wf.log.Error("could not unsubscribe from peer", logging.HostID("peerID", peerID), zap.Error(err))
			}
		}(peerID)
	}

	return nil
}

// UnsubscribeAll is used to stop receiving messages from peer(s). It does not close subscriptions
func (wf *WakuFilterPush) UnsubscribeAll(ctx context.Context, opts ...FilterUnsubscribeOption) error {
	params, err := wf.getUnsubscribeParameters(opts...)
	if err != nil {
		return err
	}

	wf.subscriptions.Lock()
	defer wf.subscriptions.Unlock()

	wf.wg.Add(len(wf.subscriptions.items))
	for peerID := range wf.subscriptions.items {
		if !params.unsubscribeAll && peerID != params.selectedPeer {
			continue
		}

		go func(peerID peer.ID) {
			defer wf.wg.Done()
			err := wf.request(
				ctx,
				&FilterSubscribeParameters{selectedPeer: peerID},
				pb.FilterSubscribeRequest_UNSUBSCRIBE_ALL,
				ContentFilter{})
			if err != nil {
				wf.log.Error("could not unsubscribe from peer", logging.HostID("peerID", peerID), zap.Error(err))
			}
		}(peerID)
	}

	return nil
}
