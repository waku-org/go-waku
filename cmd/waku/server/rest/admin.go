package rest

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/waku-org/go-waku/cmd/waku/server"
	"github.com/waku-org/go-waku/waku/v2/node"
)

type AdminService struct {
	node *node.WakuNode
	mux  *chi.Mux
}

type WakuPeer struct {
	ID         string   `json:"id"`
	MultiAddrs []string `json:"multiaddrs"`
	Protocols  []string `json:"protocols"`
	Connected  bool     `json:"connected"`
}

const routeAdminV1Peers = "/admin/v1/peers"

func NewAdminService(node *node.WakuNode, m *chi.Mux) *AdminService {
	d := &AdminService{
		node: node,
		mux:  m,
	}

	m.Get(routeAdminV1Peers, d.getV1Peers)
	//m.Post(routeAdminV1Peers, d.postV1Peer)

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
			ID:        peer.ID.Pretty(),
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

/*
func (a *AdminService) postV1Peer(w http.ResponseWriter, req *http.Request) {

	//writeErrOrResponse(w, nil, response)
} */
