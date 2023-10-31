package rest

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/waku-org/go-waku/cmd/waku/server"
	"github.com/waku-org/go-waku/logging"
	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-waku/waku/v2/peerstore"
	waku_proto "github.com/waku-org/go-waku/waku/v2/protocol"
	"go.uber.org/zap"
)

type AdminService struct {
	node *node.WakuNode
	mux  *chi.Mux
	log  *zap.Logger
}

type WakuPeer struct {
	ID         peer.ID  `json:"id"`
	MultiAddrs []string `json:"multiaddrs"`
	Protocols  []string `json:"protocols"`
	Connected  bool     `json:"connected"`
}

type WakuPeerInfo struct {
	MultiAddr string   `json:"multiaddr"`
	Shards    []int    `json:"shards"`
	Protocols []string `json:"protocols"`
}

const routeAdminV1Peers = "/admin/v1/peers"

func NewAdminService(node *node.WakuNode, m *chi.Mux, log *zap.Logger) *AdminService {
	d := &AdminService{
		node: node,
		mux:  m,
		log:  log,
	}

	m.Get(routeAdminV1Peers, d.getV1Peers)
	m.Post(routeAdminV1Peers, d.postV1Peer)

	return d
}

func (a *AdminService) getV1Peers(w http.ResponseWriter, req *http.Request) {
	peers, err := a.node.Peers()
	if err != nil {
		writeErrOrResponse(w, err, nil)
		return
	}
	response := make([]WakuPeer, 0)
	for _, peer := range peers {
		wPeer := WakuPeer{
			ID:        peer.ID,
			Connected: peer.Connected,
		}

		for _, addr := range peer.Addrs {
			wPeer.MultiAddrs = append(wPeer.MultiAddrs, addr.String())
		}
		for _, proto := range peer.Protocols {
			if !server.IsWakuProtocol(proto) {
				continue
			}
			wPeer.Protocols = append(wPeer.Protocols, string(proto))
		}
		response = append(response, wPeer)
	}

	writeErrOrResponse(w, nil, response)
}

func (a *AdminService) postV1Peer(w http.ResponseWriter, req *http.Request) {
	var pInfo WakuPeerInfo
	var topics []string
	var protos []protocol.ID

	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&pInfo); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	addr, err := ma.NewMultiaddr(pInfo.MultiAddr)
	if err != nil {
		a.log.Error("building multiaddr", zap.Error(err))
		writeErrOrResponse(w, err, nil)
		return
	}

	for _, shard := range pInfo.Shards {
		topic := waku_proto.NewStaticShardingPubsubTopic(waku_proto.ClusterIndex, uint16(shard))
		topics = append(topics, topic.String())
	}

	id, err := a.node.AddPeer(addr, peerstore.Static, topics, protos...)
	if err != nil {
		a.log.Error("failed to add peer", zap.Error(err))
		writeErrOrResponse(w, err, nil)
		return
	}
	a.log.Info("add peer successful", logging.HostID("peerID", id))
	pi := peer.AddrInfo{ID: id, Addrs: []ma.Multiaddr{addr}}
	err = a.node.Host().Connect(req.Context(), pi)
	if err != nil {
		a.log.Error("failed to connect to peer", logging.HostID("peerID", id), zap.Error(err))
		writeErrOrResponse(w, err, nil)
		return
	}
	writeErrOrResponse(w, nil, nil)
}
