package rest

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-zerokit-rln/rln"
)

type DebugService struct {
	node *node.WakuNode
	mux  *chi.Mux
}

type InfoArgs struct {
}

type InfoReply struct {
	ENRUri          string   `json:"enrUri,omitempty"`
	ListenAddresses []string `json:"listenAddresses,omitempty"`
}

// To quickly run in sandbox machine:
// nohup ./build/waku --rest --rest-address=0.0.0.0 --rest-port=30304 --rln-relay=true --rln-relay-dynamic=true --rln-relay-eth-client-address=wss://sepolia.infura.io/ws/v3/4576482c0f474483ac709755f2663b20 --rln-relay-eth-contract-address=0xF471d71E9b1455bBF4b85d475afb9BB0954A29c4 > logs.out &

// Example usage (replace by your public commitment)
// curl http://localhost:30304/debug/v1/merkleProof/15506699537643273163469326218249028331014943998871371946527772121417127479348
// curl http://65.21.94.244:30304/debug/v1/merkleProof/15506699537643273163469326218249028331014943998871371946527772121417127479348

const routeDebugInfoV1 = "/debug/v1/info"
const routeDebugVersionV1 = "/debug/v1/version"
const routeDebugGetMerkleProofV1 = "/debug/v1/merkleProof/{commitment}"

func NewDebugService(node *node.WakuNode, m *chi.Mux) *DebugService {
	d := &DebugService{
		node: node,
		mux:  m,
	}

	m.Get(routeDebugInfoV1, d.getV1Info)
	m.Get(routeDebugVersionV1, d.getV1Version)
	m.Get(routeDebugGetMerkleProofV1, d.getV1MerkleProof)

	return d
}

type VersionResponse string

func (d *DebugService) getV1Info(w http.ResponseWriter, req *http.Request) {
	response := new(InfoReply)
	response.ENRUri = d.node.ENR().String()
	for _, addr := range d.node.ListenAddresses() {
		response.ListenAddresses = append(response.ListenAddresses, addr.String())
	}
	writeErrOrResponse(w, nil, response)
}

func (d *DebugService) getV1Version(w http.ResponseWriter, req *http.Request) {
	response := VersionResponse(node.GetVersionInfo().String())
	writeErrOrResponse(w, nil, response)
}

type MerkleProofResponse struct {
	MerkleRoot        string   `json:"root"`
	MerkePathElements []string `json:"pathElements"`
	MerkePathIndexes  []uint8  `json:"pathIndexes"`
	LeafIndex         uint64   `json:"leafIndex"`
	CommitmentId      string   `json:"commitmentId"`
}

func (d *DebugService) getV1MerkleProof(w http.ResponseWriter, req *http.Request) {
	commitmentReq := chi.URLParam(req, "commitment")
	if commitmentReq == "" {
		writeErrResponse(w, nil, fmt.Errorf("commitment is empty"), http.StatusBadRequest)
		return
	}

	commitmentReqBig := new(big.Int)
	commitmentReqBig, ok := commitmentReqBig.SetString(commitmentReq, 10)
	if !ok {
		writeErrResponse(w, nil, fmt.Errorf("commitment is not a number"), http.StatusBadRequest)
		return
	}

	fmt.Println("commitmentReq: ", commitmentReq)

	// Some dirty way to access rln and group manager
	rlnInstance := d.node.RLNInstance
	groupManager := d.node.GroupManager

	ready, err := groupManager.IsReady(context.Background())
	if err != nil {
		writeErrResponse(w, nil, fmt.Errorf("could not check if service was ready"), http.StatusBadRequest)
		return
	}
	if !ready {
		writeErrResponse(w, nil, fmt.Errorf("service is not ready"), http.StatusBadRequest)
		return
	}

	found := false
	membershipIndex := uint(0)
	for leafIdx := uint(0); leafIdx < rlnInstance.LeavesSet(); leafIdx++ {

		leaf, err := rlnInstance.GetLeaf(leafIdx)
		if err != nil {
			writeErrResponse(w, nil, fmt.Errorf("could not get leaf"), http.StatusBadRequest)
			return
		}

		leafBig := rln.Bytes32ToBigInt(leaf)

		if leafBig.Cmp(commitmentReqBig) == 0 {
			found = true
			membershipIndex = leafIdx
			break
		}
	}
	if !found {
		writeErrResponse(w, nil, fmt.Errorf("commitment not found: %s", commitmentReqBig.String()), http.StatusBadRequest)
		return
	}

	fmt.Println("membershipIndex: ", membershipIndex)

	merkleProof, err := rlnInstance.GetMerkleProof(rln.MembershipIndex(membershipIndex))
	if err != nil {
		writeErrOrResponse(w, err, nil)
		return
	}

	elementsStr := make([]string, 0)
	indexesStr := make([]uint8, 0)

	for _, path := range merkleProof.PathElements {
		fmt.Println("path: ", path)
		elementsStr = append(elementsStr, hex.EncodeToString(path[:]))
	}
	for _, index := range merkleProof.PathIndexes {
		indexesStr = append(indexesStr, uint8(index))
	}

	// TODO: Not nide to get proof and root in different non atomic calls. In an unlikely edge case the tree can change between the two calls
	// if a membership is added. Proof of concept by now.
	merkleRoot, err := rlnInstance.GetMerkleRoot()
	if err != nil {
		writeErrOrResponse(w, err, nil)
		return
	}

	fmt.Println("merkleRoot: ", merkleRoot)

	writeErrOrResponse(w, nil, MerkleProofResponse{
		MerkleRoot:        hex.EncodeToString(merkleRoot[:]),
		MerkePathElements: elementsStr,
		MerkePathIndexes:  indexesStr,
		LeafIndex:         uint64(membershipIndex),
		CommitmentId:      commitmentReqBig.String(),
	})
}
