package lightpush

import (
	"context"
	"encoding/hex"
	"errors"
	"math"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	libp2pProtocol "github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-msgio/pbio"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/waku-org/go-waku/logging"
	"github.com/waku-org/go-waku/waku/v2/peermanager"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/lightpush/pb"
	wpb "github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"go.uber.org/zap"
)

// LightPushID_v20beta1 is the current Waku LightPush protocol identifier
const LightPushID_v20beta1 = libp2pProtocol.ID("/vac/waku/lightpush/2.0.0-beta1")

var (
	ErrNoPeersAvailable = errors.New("no suitable remote peers")
	ErrInvalidID        = errors.New("invalid request id")
)

// WakuLightPush is the implementation of the Waku LightPush protocol
type WakuLightPush struct {
	h       host.Host
	relay   *relay.WakuRelay
	cancel  context.CancelFunc
	pm      *peermanager.PeerManager
	metrics Metrics

	log *zap.Logger
}

// NewWakuLightPush returns a new instance of Waku Lightpush struct
// Takes an optional peermanager if WakuLightPush is being created along with WakuNode.
// If using libp2p host, then pass peermanager as nil
func NewWakuLightPush(relay *relay.WakuRelay, pm *peermanager.PeerManager, reg prometheus.Registerer, log *zap.Logger) *WakuLightPush {
	wakuLP := new(WakuLightPush)
	wakuLP.relay = relay
	wakuLP.log = log.Named("lightpush")
	wakuLP.pm = pm
	wakuLP.metrics = newMetrics(reg)
	return wakuLP
}

// Sets the host to be able to mount or consume a protocol
func (wakuLP *WakuLightPush) SetHost(h host.Host) {
	wakuLP.h = h
}

// Start inits the lighpush protocol
func (wakuLP *WakuLightPush) Start(ctx context.Context) error {
	if wakuLP.relayIsNotAvailable() {
		return errors.New("relay is required, without it, it is only a client and cannot be started")
	}

	ctx, cancel := context.WithCancel(ctx)

	wakuLP.cancel = cancel
	wakuLP.h.SetStreamHandlerMatch(LightPushID_v20beta1, protocol.PrefixTextMatch(string(LightPushID_v20beta1)), wakuLP.onRequest(ctx))
	wakuLP.log.Info("Light Push protocol started")

	return nil
}

// relayIsNotAvailable determines if this node supports relaying messages for other lightpush clients
func (wakuLP *WakuLightPush) relayIsNotAvailable() bool {
	return wakuLP.relay == nil
}

func (wakuLP *WakuLightPush) onRequest(ctx context.Context) func(network.Stream) {
	return func(stream network.Stream) {
		logger := wakuLP.log.With(logging.HostID("peer", stream.Conn().RemotePeer()))
		requestPushRPC := &pb.PushRPC{}

		writer := pbio.NewDelimitedWriter(stream)
		reader := pbio.NewDelimitedReader(stream, math.MaxInt32)

		err := reader.ReadMsg(requestPushRPC)
		if err != nil {
			logger.Error("reading request", zap.Error(err))
			wakuLP.metrics.RecordError(decodeRPCFailure)
			if err := stream.Reset(); err != nil {
				wakuLP.log.Error("resetting connection", zap.Error(err))
			}
			return
		}

		logger = logger.With(zap.String("requestID", requestPushRPC.RequestId))

		responsePushRPC := &pb.PushRPC{}
		responsePushRPC.RequestId = requestPushRPC.RequestId

		if requestPushRPC.Query != nil {
			logger.Info("push request")
			response := new(pb.PushResponse)

			pubSubTopic := requestPushRPC.Query.PubsubTopic
			message := requestPushRPC.Query.Message

			wakuLP.metrics.RecordMessage()

			// TODO: Assumes success, should probably be extended to check for network, peers, etc
			// It might make sense to use WithReadiness option here?

			_, err := wakuLP.relay.PublishToTopic(ctx, message, pubSubTopic)

			if err != nil {
				logger.Error("publishing message", zap.Error(err))
				wakuLP.metrics.RecordError(messagePushFailure)
				response.Info = "Could not publish message"
			} else {
				response.IsSuccess = true
				response.Info = "OK"
			}

			responsePushRPC.Response = response

			err = writer.WriteMsg(responsePushRPC)
			if err != nil {
				wakuLP.metrics.RecordError(writeResponseFailure)
				logger.Error("writing response", zap.Error(err))
				if err := stream.Reset(); err != nil {
					wakuLP.log.Error("resetting connection", zap.Error(err))
				}
				return
			}

			logger.Info("response sent")
			stream.Close()
		} else {
			wakuLP.metrics.RecordError(emptyRequestBodyFailure)
			if err := stream.Reset(); err != nil {
				wakuLP.log.Error("resetting connection", zap.Error(err))
			}
		}

		if requestPushRPC.Response != nil {
			if requestPushRPC.Response.IsSuccess {
				logger.Info("request success")
			} else {
				logger.Info("request failure", zap.String("info=", requestPushRPC.Response.Info))
			}
		} else {
			wakuLP.metrics.RecordError(emptyResponseBodyFailure)
		}
	}
}

// request sends a message via lightPush protocol to either a specified peer or peer that is selected.
func (wakuLP *WakuLightPush) request(ctx context.Context, req *pb.PushRequest, params *lightPushParameters) (*pb.PushResponse, error) {
	if params == nil {
		return nil, errors.New("lightpush params are mandatory")
	}

	if len(params.requestID) == 0 {
		return nil, ErrInvalidID
	}

	logger := wakuLP.log.With(logging.HostID("peer", params.selectedPeer))

	stream, err := wakuLP.h.NewStream(ctx, params.selectedPeer, LightPushID_v20beta1)
	if err != nil {
		logger.Error("creating stream to peer", zap.Error(err))
		wakuLP.metrics.RecordError(dialFailure)
		return nil, err
	}
	pushRequestRPC := &pb.PushRPC{RequestId: hex.EncodeToString(params.requestID), Query: req}

	writer := pbio.NewDelimitedWriter(stream)
	reader := pbio.NewDelimitedReader(stream, math.MaxInt32)

	err = writer.WriteMsg(pushRequestRPC)
	if err != nil {
		wakuLP.metrics.RecordError(writeRequestFailure)
		logger.Error("writing request", zap.Error(err))
		if err := stream.Reset(); err != nil {
			wakuLP.log.Error("resetting connection", zap.Error(err))
		}
		return nil, err
	}

	pushResponseRPC := &pb.PushRPC{}
	err = reader.ReadMsg(pushResponseRPC)
	if err != nil {
		logger.Error("reading response", zap.Error(err))
		wakuLP.metrics.RecordError(decodeRPCFailure)
		if err := stream.Reset(); err != nil {
			wakuLP.log.Error("resetting connection", zap.Error(err))
		}
		return nil, err
	}

	stream.Close()

	return pushResponseRPC.Response, nil
}

// Stop unmounts the lightpush protocol
func (wakuLP *WakuLightPush) Stop() {
	if wakuLP.cancel == nil {
		return
	}

	wakuLP.cancel()
	wakuLP.h.RemoveStreamHandler(LightPushID_v20beta1)
}

func (wakuLP *WakuLightPush) handleOpts(ctx context.Context, message *wpb.WakuMessage, opts ...Option) (*lightPushParameters, error) {
	params := new(lightPushParameters)
	params.host = wakuLP.h
	params.log = wakuLP.log
	params.pm = wakuLP.pm
	var err error

	optList := append(DefaultOptions(wakuLP.h), opts...)
	for _, opt := range optList {
		opt(params)
	}

	if params.pubsubTopic == "" {
		params.pubsubTopic, err = protocol.GetPubSubTopicFromContentTopic(message.ContentTopic)
		if err != nil {
			return nil, err
		}
	}

	if params.pm != nil && params.selectedPeer == "" {
		params.selectedPeer, err = wakuLP.pm.SelectPeer(
			peermanager.PeerSelectionCriteria{
				SelectionType: params.peerSelectionType,
				Proto:         LightPushID_v20beta1,
				PubsubTopic:   params.pubsubTopic,
				SpecificPeers: params.preferredPeers,
				Ctx:           ctx,
			},
		)
	}
	if params.selectedPeer == "" {
		if err != nil {
			params.log.Error("selecting peer", zap.Error(err))
			wakuLP.metrics.RecordError(peerNotFoundFailure)
			return nil, ErrNoPeersAvailable
		}
	}
	return params, nil
}

// Optional PublishToTopic is used to broadcast a WakuMessage to a pubsub topic via lightpush protocol
// If pubSubTopic is not provided, then contentTopic is use to derive the relevant pubSubTopic via autosharding.
func (wakuLP *WakuLightPush) PublishToTopic(ctx context.Context, message *wpb.WakuMessage, opts ...Option) ([]byte, error) {
	if message == nil {
		return nil, errors.New("message can't be null")
	}

	params, err := wakuLP.handleOpts(ctx, message, opts...)
	if err != nil {
		return nil, err
	}
	req := new(pb.PushRequest)
	req.Message = message
	req.PubsubTopic = params.pubsubTopic

	response, err := wakuLP.request(ctx, req, params)
	if err != nil {
		return nil, err
	}

	if response.IsSuccess {
		hash := message.Hash(params.pubsubTopic)
		wakuLP.log.Info("waku.lightpush published", logging.HexString("hash", hash))
		return hash, nil
	}

	return nil, errors.New(response.Info)
}

// Publish is used to broadcast a WakuMessage to the pubSubTopic (which is derived from the contentTopic) via lightpush protocol
// If auto-sharding is not to be used, then PublishToTopic API should be used
func (wakuLP *WakuLightPush) Publish(ctx context.Context, message *wpb.WakuMessage, opts ...Option) ([]byte, error) {
	return wakuLP.PublishToTopic(ctx, message, opts...)
}
