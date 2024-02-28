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

// DefaultPageSize is the default number of waku messages per page
const DefaultPageSize = 20

var (

	// ErrNoPeersAvailable is returned when there are no store peers in the peer store
	// that could be used to retrieve message history
	ErrNoPeersAvailable = errors.New("no suitable remote peers")
	ErrMustSelectPeer   = errors.New("a peer ID or multiaddress is required")
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

	filterCriteria, isFilterCriteria := criteria.(FilterCriteria)

	var pubsubTopics []string
	if isFilterCriteria {
		pubsubTopics = append(pubsubTopics, filterCriteria.PubsubTopic)
	}

	//Add Peer to peerstore.
	if s.pm != nil && params.peerAddr != nil {
		pData, err := s.pm.AddPeer(params.peerAddr, peerstore.Static, pubsubTopics, StoreID_v300)
		if err != nil {
			return nil, err
		}
		s.pm.Connect(pData)
		params.selectedPeer = pData.AddrInfo.ID
	}

	if s.pm != nil && params.selectedPeer == "" {
		if isFilterCriteria {
			selectedPeers, err := s.pm.SelectPeers(
				peermanager.PeerSelectionCriteria{
					SelectionType: params.peerSelectionType,
					Proto:         StoreID_v300,
					PubsubTopics:  []string{filterCriteria.PubsubTopic},
					SpecificPeers: params.preferredPeers,
					Ctx:           ctx,
				},
			)
			if err != nil {
				return nil, err
			}
			params.selectedPeer = selectedPeers[0]
		} else {
			return nil, ErrMustSelectPeer
		}
	}

	if params.selectedPeer == "" {
		return nil, ErrNoPeersAvailable
	}

	pageLimit := params.pageLimit
	if pageLimit == 0 {
		pageLimit = DefaultPageSize
	} else if pageLimit > uint64(MaxPageSize) {
		pageLimit = MaxPageSize
	}

	storeRequest := &pb.StoreRequest{
		RequestId:         hex.EncodeToString(params.requestID),
		ReturnValues:      params.returnValues,
		PaginationForward: params.forward,
		PaginationLimit:   proto.Uint64(pageLimit),
	}

	criteria.PopulateStoreRequest(storeRequest)

	if params.cursor != nil {
		storeRequest.PaginationCursor = params.cursor
	}

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
		cursor:       response.PaginationCursor,
	}

	return result, nil
}

func (s *WakuStoreV3) Retrieve(ctx context.Context, messageHashes []wpb.MessageHash, opts ...RequestOption) (*Result, error) {
	opts = append(opts, WithReturnValues(true))
	return s.Request(ctx, MessageHashCriteria{messageHashes}, opts...)
}

// Exists is used to determine if a set of message hashes exist. Also returns a cursor in case there is more than one page of results
func (s *WakuStoreV3) Exists(ctx context.Context, messageHashes []wpb.MessageHash, opts ...RequestOption) (map[wpb.MessageHash]bool, []byte, error) {
	opts = append(opts, WithReturnValues(false))
	result, err := s.Request(ctx, MessageHashCriteria{messageHashes}, opts...)
	if err != nil {
		return nil, nil, err
	}

	msgMap := make(map[wpb.MessageHash]bool)
	for i, _ := range messageHashes {
		msgMap[messageHashes[i]] = false
	}

	for _, m := range result.Messages {
		h := wpb.ToMessageHash(m.MessageHash)
		msgMap[h] = true
	}

	return msgMap, result.cursor, nil
}

func (s *WakuStoreV3) Next(ctx context.Context, r *Result) (*Result, error) {
	if r.IsComplete() {
		return &Result{
			store:        s,
			started:      true,
			Messages:     []*pb.WakuMessageKeyValue{},
			cursor:       nil,
			storeRequest: r.storeRequest,
			peerID:       r.PeerID(),
		}, nil
	}

	storeRequest := proto.Clone(r.storeRequest).(*pb.StoreRequest)
	storeRequest.RequestId = hex.EncodeToString(protocol.GenerateRequestID())
	storeRequest.PaginationCursor = r.Cursor()

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
		cursor:       response.PaginationCursor,
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

	storeResponse := &pb.StoreResponse{RequestId: storeRequest.RequestId}
	err = reader.ReadMsg(storeResponse)
	if err != nil {
		logger.Error("reading response", zap.Error(err))
		if err := stream.Reset(); err != nil {
			s.log.Error("resetting connection", zap.Error(err))
		}
		return nil, err
	}

	stream.Close()

	if err := storeResponse.Validate(storeRequest.RequestId); err != nil {
		return nil, err
	}

	// TODO: validate error codes

	return storeResponse, nil
}
