package store

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/cruxic/go-hmac-drbg/hmacdrbg"
	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	libp2pProtocol "github.com/libp2p/go-libp2p-core/protocol"
	"github.com/libp2p/go-msgio/protoio"

	ma "github.com/multiformats/go-multiaddr"
	"github.com/status-im/go-waku/waku/common"
	"github.com/status-im/go-waku/waku/v2/protocol"
)

var log = logging.Logger("wakustore")

const WakuStoreProtocolId = libp2pProtocol.ID("/vac/waku/store/2.0.0-beta1")
const MaxPageSize = 100 // Maximum number of waku messages in each page
const ConnectionTimeout = 10 * time.Second
const DefaultContentTopic = "/waku/2/default-content/proto"

func minOf(vars ...int) int {
	min := vars[0]

	for _, i := range vars {
		if min > i {
			min = i
		}
	}

	return min
}

func paginateWithIndex(list []IndexedWakuMessage, pinfo *protocol.PagingInfo) (resMessages []IndexedWakuMessage, resPagingInfo *protocol.PagingInfo) {
	// takes list, and performs paging based on pinfo
	// returns the page i.e, a sequence of IndexedWakuMessage and the new paging info to be used for the next paging request
	cursor := pinfo.Cursor
	pageSize := pinfo.PageSize
	dir := pinfo.Direction

	if pageSize == 0 { // pageSize being zero indicates that no pagination is required
		return list, pinfo
	}

	if len(list) == 0 { // no pagination is needed for an empty list
		return list, &protocol.PagingInfo{PageSize: 0, Cursor: pinfo.Cursor, Direction: pinfo.Direction}
	}

	msgList := make([]IndexedWakuMessage, len(list))
	_ = copy(msgList, list) // makes a copy of the list

	sort.Slice(msgList, func(i, j int) bool { // sorts msgList based on the custom comparison proc indexedWakuMessageComparison
		return indexedWakuMessageComparison(msgList[i], msgList[j]) == -1
	})

	initQuery := false
	if cursor == nil {
		initQuery = true // an empty cursor means it is an initial query
		switch dir {
		case protocol.PagingInfo_FORWARD:
			cursor = list[0].index // perform paging from the begining of the list
		case protocol.PagingInfo_BACKWARD:
			cursor = list[len(list)-1].index // perform paging from the end of the list
		}
	}

	foundIndex := findIndex(msgList, cursor)
	if foundIndex == -1 { // the cursor is not valid
		return nil, &protocol.PagingInfo{PageSize: 0, Cursor: pinfo.Cursor, Direction: pinfo.Direction}
	}

	var retrievedPageSize, s, e int
	var newCursor *protocol.Index // to be returned as part of the new paging info
	switch dir {
	case protocol.PagingInfo_FORWARD: // forward pagination
		remainingMessages := len(msgList) - foundIndex - 1
		if initQuery {
			remainingMessages = remainingMessages + 1
			foundIndex = foundIndex - 1
		}
		// the number of queried messages cannot exceed the MaxPageSize and the total remaining messages i.e., msgList.len-foundIndex
		retrievedPageSize = minOf(int(pageSize), MaxPageSize, remainingMessages)
		s = foundIndex + 1 // non inclusive
		e = foundIndex + retrievedPageSize
		newCursor = msgList[e].index // the new cursor points to the end of the page
	case protocol.PagingInfo_BACKWARD: // backward pagination
		remainingMessages := foundIndex
		if initQuery {
			remainingMessages = remainingMessages + 1
			foundIndex = foundIndex + 1
		}
		// the number of queried messages cannot exceed the MaxPageSize and the total remaining messages i.e., foundIndex-0
		retrievedPageSize = minOf(int(pageSize), MaxPageSize, remainingMessages)
		s = foundIndex - retrievedPageSize
		e = foundIndex - 1
		newCursor = msgList[s].index // the new cursor points to the begining of the page
	}

	// retrieve the messages
	for i := s; i <= e; i++ {
		resMessages = append(resMessages, msgList[i])
	}
	resPagingInfo = &protocol.PagingInfo{PageSize: uint64(retrievedPageSize), Cursor: newCursor, Direction: pinfo.Direction}

	return
}

func paginateWithoutIndex(list []IndexedWakuMessage, pinfo *protocol.PagingInfo) (resMessages []*protocol.WakuMessage, resPinfo *protocol.PagingInfo) {
	// takes list, and performs paging based on pinfo
	// returns the page i.e, a sequence of WakuMessage and the new paging info to be used for the next paging request
	indexedData, updatedPagingInfo := paginateWithIndex(list, pinfo)
	for _, indexedMsg := range indexedData {
		resMessages = append(resMessages, indexedMsg.msg)
	}
	resPinfo = updatedPagingInfo
	return
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func (w *WakuStore) FindMessages(query *protocol.HistoryQuery) *protocol.HistoryResponse {
	result := new(protocol.HistoryResponse)
	// data holds IndexedWakuMessage whose topics match the query
	var data []IndexedWakuMessage
	for _, indexedMsg := range w.messages {
		if contains(query.Topics, indexedMsg.msg.ContentTopic) {
			data = append(data, indexedMsg)
		}
	}

	result.Messages, result.PagingInfo = paginateWithoutIndex(data, query.PagingInfo)
	return result
}

type MessageProvider interface {
	GetAll() ([]*protocol.WakuMessage, error)
	Put(cursor *protocol.Index, message *protocol.WakuMessage) error
	Stop()
}

type IndexedWakuMessage struct {
	msg   *protocol.WakuMessage
	index *protocol.Index
}

type WakuStore struct {
	msg           chan *common.Envelope
	messages      []IndexedWakuMessage
	messagesMutex sync.Mutex

	msgProvider MessageProvider
	h           host.Host
	ctx         context.Context
}

func NewWakuStore(ctx context.Context, h host.Host, msg chan *common.Envelope, p MessageProvider) *WakuStore {
	wakuStore := new(WakuStore)
	wakuStore.msg = msg
	wakuStore.msgProvider = p
	wakuStore.h = h
	wakuStore.ctx = ctx

	return wakuStore
}

func (store *WakuStore) Start() {
	if store.msgProvider == nil {
		return
	}

	store.h.SetStreamHandler(WakuStoreProtocolId, store.onRequest)

	messages, err := store.msgProvider.GetAll()
	if err != nil {
		log.Error("could not load DBProvider messages")
		return
	}

	for _, msg := range messages {
		idx, err := computeIndex(msg)
		if err != nil {
			log.Error("could not calculate message index", err)
			continue
		}
		store.messages = append(store.messages, IndexedWakuMessage{msg: msg, index: idx})
	}

	go store.storeIncomingMessages()
}

func (store *WakuStore) Stop() {
	store.msgProvider.Stop()
}

func (store *WakuStore) storeIncomingMessages() {
	for envelope := range store.msg {
		index, err := computeIndex(envelope.Message())
		if err != nil {
			log.Error("could not calculate message index", err)
			continue
		}

		store.messagesMutex.Lock()
		store.messages = append(store.messages, IndexedWakuMessage{msg: envelope.Message(), index: index})
		store.messagesMutex.Unlock()

		if store.msgProvider == nil {
			continue
		}

		err = store.msgProvider.Put(index, envelope.Message()) // Should the index be stored?
		if err != nil {
			log.Error("could not store message", err)
			continue
		}
	}
}

func (store *WakuStore) onRequest(s network.Stream) {
	defer s.Close()

	historyRPCRequest := &protocol.HistoryRPC{}

	writer := protoio.NewDelimitedWriter(s)
	reader := protoio.NewDelimitedReader(s, 64*1024)

	err := reader.ReadMsg(historyRPCRequest)
	if err != nil {
		log.Error("error reading request", err)
		return
	}

	log.Info(fmt.Sprintf("%s: Received query from %s", s.Conn().LocalPeer(), s.Conn().RemotePeer()))

	historyResponseRPC := &protocol.HistoryRPC{}
	historyResponseRPC.RequestId = historyRPCRequest.RequestId
	historyResponseRPC.Response = store.FindMessages(historyRPCRequest.Query)

	err = writer.WriteMsg(historyResponseRPC)
	if err != nil {
		log.Error("error writing response", err)
		s.Reset()
	} else {
		log.Info(fmt.Sprintf("%s: Response sent  to %s", s.Conn().LocalPeer().String(), s.Conn().RemotePeer().String()))
	}
}

func computeIndex(msg *protocol.WakuMessage) (*protocol.Index, error) {
	data, err := msg.Marshal()
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(data)
	return &protocol.Index{
		Digest:       digest[:],
		ReceivedTime: float64(time.Now().UnixNano()),
	}, nil
}

func indexComparison(x, y *protocol.Index) int {
	// compares x and y
	// returns 0 if they are equal
	// returns -1 if x < y
	// returns 1 if x > y

	var timecmp int = 0 // TODO: ask waku team why Index ReceivedTime is is float?
	if x.ReceivedTime > y.ReceivedTime {
		timecmp = 1
	} else if x.ReceivedTime < y.ReceivedTime {
		timecmp = -1
	}

	digestcm := bytes.Compare(x.Digest, y.Digest)
	if timecmp != 0 {
		return timecmp // timestamp has a higher priority for comparison
	}

	return digestcm
}

func indexedWakuMessageComparison(x, y IndexedWakuMessage) int {
	// compares x and y
	// returns 0 if they are equal
	// returns -1 if x < y
	// returns 1 if x > y
	return indexComparison(x.index, y.index)
}

func findIndex(msgList []IndexedWakuMessage, index *protocol.Index) int {
	// returns the position of an IndexedWakuMessage in msgList whose index value matches the given index
	// returns -1 if no match is found
	for i, indexedWakuMessage := range msgList {
		if bytes.Compare(indexedWakuMessage.index.Digest, index.Digest) == 0 && indexedWakuMessage.index.ReceivedTime == index.ReceivedTime {
			return i
		}
	}
	return -1
}

func (store *WakuStore) AddPeer(p peer.ID, addrs []ma.Multiaddr) error {
	for _, addr := range addrs {
		store.h.Peerstore().AddAddr(p, addr, peerstore.PermanentAddrTTL)
	}
	err := store.h.Peerstore().AddProtocols(p, string(WakuStoreProtocolId))
	if err != nil {
		return err
	}
	return nil
}

func (store *WakuStore) selectPeer() *peer.ID {
	// @TODO We need to be more stratigic about which peers we dial. Right now we just set one on the service.
	// Ideally depending on the query and our set  of peers we take a subset of ideal peers.
	// This will require us to check for various factors such as:
	//  - which topics they track
	//  - latency?
	//  - default store peer?

	// Selects the best peer for a given protocol
	var peers peer.IDSlice
	for _, peer := range store.h.Peerstore().Peers() {
		protocols, err := store.h.Peerstore().SupportsProtocols(peer, string(WakuStoreProtocolId))
		if err != nil {
			log.Error("error obtaining the protocols supported by peers", err)
			return nil
		}

		if len(protocols) > 0 {
			peers = append(peers, peer)
		}
	}

	if len(peers) >= 1 {
		// TODO: proper heuristic here that compares peer scores and selects "best" one. For now the first peer for the given protocol is returned
		return &peers[0]
	}

	return nil
}

var brHmacDrbgPool = sync.Pool{New: func() interface{} {
	seed := make([]byte, 48)
	_, err := rand.Read(seed)
	if err != nil {
		log.Fatal(err)
	}
	return hmacdrbg.NewHmacDrbg(256, seed, nil)
}}

func GenerateRequestId() string {
	rng := brHmacDrbgPool.Get().(*hmacdrbg.HmacDrbg)
	defer brHmacDrbgPool.Put(rng)

	randData := make([]byte, 10)
	if !rng.Generate(randData) {
		//Reseed is required every 10,000 calls
		seed := make([]byte, 48)
		_, err := rand.Read(seed)
		if err != nil {
			log.Fatal(err)
		}
		err = rng.Reseed(seed)
		if err != nil {
			//only happens if seed < security-level
			log.Fatal(err)
		}

		if !rng.Generate(randData) {
			log.Error("could not generate random request id")
		}
	}
	return hex.EncodeToString(randData)
}

func (store *WakuStore) Query(q *protocol.HistoryQuery) (*protocol.HistoryResponse, error) {
	peer := store.selectPeer()
	if peer == nil {
		return nil, errors.New("no suitable remote peers")
	}

	ctx, cancel := context.WithTimeout(store.ctx, ConnectionTimeout)
	defer cancel()

	connOpt, err := store.h.NewStream(ctx, *peer, WakuStoreProtocolId)
	if err != nil {
		log.Info("failed to connect to remote peer", err)
		return nil, err
	}

	defer connOpt.Close()
	defer connOpt.Reset()

	historyRequest := &protocol.HistoryRPC{Query: q, RequestId: GenerateRequestId()}

	writer := protoio.NewDelimitedWriter(connOpt)
	reader := protoio.NewDelimitedReader(connOpt, 64*1024)

	err = writer.WriteMsg(historyRequest)
	if err != nil {
		log.Error("could not write request", err)
		return nil, err
	}

	historyResponseRPC := &protocol.HistoryRPC{}
	err = reader.ReadMsg(historyResponseRPC)
	if err != nil {
		log.Error("could not read response", err)
		return nil, err
	}

	return historyResponseRPC.Response, nil
}

// TODO: queryWithAccounting
