package rpc

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/status-im/go-waku/waku/v2/node"
)

type DebugService struct {
	node *node.WakuNode
}

type InfoArgs struct {
}

type InfoReply struct {
	ENRUri          string   `json:"enrUri,omitempty"`
	ListenAddresses []string `json:"listenAddresses,omitempty"`
}

func NewDebugService(node *node.WakuNode, m *mux.Router) *DebugService {
	d := &DebugService{
		node: node,
	}

	m.HandleFunc("/debug/v1/info", d.restGetV1Info).Methods(http.MethodGet)
	m.HandleFunc("/debug/v1/version", d.restGetV1Version).Methods(http.MethodGet)

	return d
}

func (d *DebugService) GetV1Info(r *http.Request, args *InfoArgs, reply *InfoReply) error {
	reply.ENRUri = d.node.ENR().String()
	for _, addr := range d.node.ListenAddresses() {
		reply.ListenAddresses = append(reply.ListenAddresses, addr.String())
	}
	return nil
}

type VersionResponse string

func (d *DebugService) GetV1Version(r *http.Request, args *InfoArgs, reply *VersionResponse) error {
	*reply = VersionResponse(fmt.Sprintf("%s-%s", node.Version, node.GitCommit))
	return nil
}

func (d *DebugService) restGetV1Info(w http.ResponseWriter, r *http.Request) {
	response := new(InfoReply)
	err := d.GetV1Info(r, nil, response)
	writeErrOrResponse(w, err, response)
}

func (d *DebugService) restGetV1Version(w http.ResponseWriter, r *http.Request) {
	response := new(VersionResponse)
	err := d.GetV1Version(r, nil, response)
	writeErrOrResponse(w, err, response)
}
