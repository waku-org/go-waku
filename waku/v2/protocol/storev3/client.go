package storev3

import (
	"context"
	"encoding/hex"
	"errors"
	"math"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	libp2pProtocol "github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-msgio/pbio"
	"github.com/waku-org/go-waku/logging"
	"github.com/waku-org/go-waku/waku/v2/peermanager"
	"github.com/waku-org/go-waku/waku/v2/peerstore"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	wpb "github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/storev3/pb"
	"github.com/waku-org/go-waku/waku/v2/timesource"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// StoreID_v300 is the Store protocol v3 identifier
const StoreID_v300 = libp2pProtocol.ID("/vac/waku/store/3.0.0")

// MaxPageSize is the maximum number of waku messages to return per page
const MaxPageSize = 100

const DefaultPageSize = 20

var (

	// ErrNoPeersAvailable is returned when there are no store peers in the peer store
	// that could be used to retrieve message history
	ErrNoPeersAvailable = errors.New("no suitable remote peers")
)

type WakuStoreV3 struct {
	h          host.Host
	timesource timesource.Timesource
	log        *zap.Logger
	pm         *peermanager.PeerManager
}

func NewWakuStoreV3(pm *peermanager.PeerManager, timesource timesource.Timesource, log *zap.Logger) *WakuStoreV3 {
	s := new(WakuStoreV3)
	s.log = log.Named("storev3-client")
	s.timesource = timesource
	s.pm = pm
	return s
}

// Sets the host to be able to mount or consume a protocol
func (s *WakuStoreV3) SetHost(h host.Host) {
	s.h = h
}

func (s *WakuStoreV3) Request(ctx context.Context, criteria Criteria, opts ...RequestOption) (*Result, error) {
	params := new(Parameters)

	optList := DefaultOptions()
	optList = append(optList, opts...)
	for _, opt := range optList {
		err := opt(params)
		if err != nil {
			return nil, err
		}
	}

	//Add Peer to peerstore.
	if s.pm != nil && params.peerAddr != nil {
		pData, err := s.pm.AddPeer(params.peerAddr, peerstore.Static, criteria.PubsubTopic, StoreID_v300)
		if err != nil {
			return nil, err
		}
		s.pm.Connect(pData)
		params.selectedPeer = pData.AddrInfo.ID
	}

	if s.pm != nil && params.selectedPeer == "" {
		selectedPeers, err := s.pm.SelectPeers(
			peermanager.PeerSelectionCriteria{
				SelectionType: params.peerSelectionType,
				Proto:         StoreID_v300,
				PubsubTopics:  []string{criteria.PubsubTopic},
				SpecificPeers: params.preferredPeers,
				Ctx:           ctx,
			},
		)
		if err != nil {
			return nil, err
		}
		params.selectedPeer = selectedPeers[0]
	}

	// TODO Criteria should be an interface that either allows specifying a content filter, or a set of message hashes

	storeRequest := &pb.StoreRequest{
		RequestId: hex.EncodeToString(params.requestID),

		PubsubTopic:   query.PubsubTopic,
		ContentTopics: query.ContentTopics,
		TimeStart:     query.TimeStart,
		TimeEnd:       query.TimeEnd,

		MessageHashes: query.MessageHashes,

		ReturnValues: query.ReturnValues,
	}

	storeRequest.PaginationForward = params.forward

	if params.selectedPeer == "" {
		return nil, ErrNoPeersAvailable
	}

	if params.cursor != nil {
		storeRequest.PaginationCursor = params.cursor
	}

	pageSize := params.pageSize
	if pageSize == 0 {
		pageSize = DefaultPageSize
	} else if pageSize > uint64(MaxPageSize) {
		pageSize = MaxPageSize
	}
	storeRequest.PaginationLimit = proto.Uint64(pageSize)

	err := storeRequest.Validate()
	if err != nil {
		return nil, err
	}

	response, err := s.queryFrom(ctx, storeRequest, params.selectedPeer)
	if err != nil {
		return nil, err
	}

	result := &Result{
		store:        s,
		Messages:     response.Messages,
		storeRequest: storeRequest,
		peerID:       params.selectedPeer,
	}

	if response.PaginationCursor != nil {
		result.cursor = response.PaginationCursor
	}

	return result, nil
}

func (s *WakuStoreV3) Retrieve() {
	// TODO: returns the messages
}

func (s *WakuStoreV3) Exists(envelopeHashes ...[]byte) (map[[]byte]bool, error) {
	// TODO: verify if a set of message exists
}

func (s *WakuStoreV3) Next(ctx context.Context, r *Result) (*Result, error) {
	if r.IsComplete() {
		return &Result{
			store:        s,
			started:      true,
			Messages:     []*wpb.WakuMessage{},
			cursor:       nil,
			storeRequest: r.storeRequest,
			peerID:       r.PeerID(),
		}, nil
	}

	storeRequest := &pb.StoreRequest{
		RequestId: hex.EncodeToString(protocol.GenerateRequestID()),

		PubsubTopic:   query.PubsubTopic,
		ContentTopics: query.ContentTopics,
		TimeStart:     query.TimeStart,
		TimeEnd:       query.TimeEnd,

		MessageHashes: query.MessageHashes,

		ReturnValues: query.ReturnValues,

		PaginationForward: query.PaginationForward,
		PaginationLimit:   query.PaginationLimit,

		PaginationCursor: r.Cursor(),
	}

	response, err := s.queryFrom(ctx, storeRequest, r.PeerID())
	if err != nil {
		return nil, err
	}

	result := &Result{
		started:      true,
		store:        s,
		Messages:     response.Messages,
		storeRequest: storeRequest,
		peerID:       r.PeerID(),
	}

	if response.PaginationCursor != nil {
		result.cursor = response.PaginationCursor
	}

	return result, nil

}

func (s *WakuStoreV3) queryFrom(ctx context.Context, storeRequest *pb.StoreRequest, selectedPeer peer.ID) (*pb.StoreResponse, error) {
	logger := s.log.With(logging.HostID("peer", selectedPeer))
	logger.Info("sending store request")

	stream, err := s.h.NewStream(ctx, selectedPeer, StoreID_v300)
	if err != nil {
		logger.Error("creating stream to peer", zap.Error(err))
		return nil, err
	}

	writer := pbio.NewDelimitedWriter(stream)
	reader := pbio.NewDelimitedReader(stream, math.MaxInt32)

	err = writer.WriteMsg(storeRequest)
	if err != nil {
		logger.Error("writing request", zap.Error(err))
		if err := stream.Reset(); err != nil {
			s.log.Error("resetting connection", zap.Error(err))
		}
		return nil, err
	}

	historyResponseRPC := &pb.HistoryRPC{RequestId: historyRequest.RequestId}
	err = reader.ReadMsg(historyResponseRPC)
	if err != nil {
		logger.Error("reading response", zap.Error(err))
		if err := stream.Reset(); err != nil {
			s.log.Error("resetting connection", zap.Error(err))
		}
		return nil, err
	}

	stream.Close()

	// nwaku does not return a response if there are no results due to the way their
	// protobuffer library works. this condition once they have proper proto3 support
	if historyResponseRPC.Response == nil {
		// Empty response
		return &pb.HistoryResponse{
			PagingInfo: &pb.PagingInfo{},
		}, nil
	}

	if err := historyResponseRPC.ValidateResponse(storeRequest.RequestId); err != nil {
		return nil, err
	}

	// TODO: validate error codes

	return historyResponseRPC.Response, nil
}
