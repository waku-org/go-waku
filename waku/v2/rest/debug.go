package rest

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/status-im/go-waku/waku/v2/node"
)

type DebugService struct {
	node *node.WakuNode
	mux  *mux.Router
}

type InfoArgs struct {
}

type InfoReply struct {
	ENRUri          string   `json:"enrUri,omitempty"`
	ListenAddresses []string `json:"listenAddresses,omitempty"`
}

const ROUTE_DEBUG_INFOV1 = "/debug/v1/info"
const ROUTE_DEBUG_VERSIONV1 = "/debug/v1/info"

func NewDebugService(node *node.WakuNode, m *mux.Router) *DebugService {
	d := &DebugService{
		node: node,
		mux:  m,
	}

	m.HandleFunc(ROUTE_DEBUG_INFOV1, d.getV1Info).Methods(http.MethodGet)
	m.HandleFunc(ROUTE_DEBUG_VERSIONV1, d.getV1Version).Methods(http.MethodGet)

	return d
}

type VersionResponse string

func (d *DebugService) getV1Info(w http.ResponseWriter, r *http.Request) {
	response := new(InfoReply)
	response.ENRUri = d.node.ENR().String()
	for _, addr := range d.node.ListenAddresses() {
		response.ListenAddresses = append(response.ListenAddresses, addr.String())
	}
	writeErrOrResponse(w, nil, response)
}

func (d *DebugService) getV1Version(w http.ResponseWriter, r *http.Request) {
	response := VersionResponse(fmt.Sprintf("%s-%s", node.Version, node.GitCommit))
	writeErrOrResponse(w, nil, response)
}
