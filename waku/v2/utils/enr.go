package utils

import (
	"crypto/ecdsa"
	"fmt"
	"math"
	"net"
	"strconv"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

const WakuENRField = "waku2"

// WakuEnrBitfield is a8-bit flag field to indicate Waku capabilities. Only the 4 LSBs are currently defined according to RFC31 (https://rfc.vac.dev/spec/31/).
type WakuEnrBitfield = uint8

func NewWakuEnrBitfield(lightpush, filter, store, relay bool) WakuEnrBitfield {
	var v uint8 = 0

	if lightpush {
		v |= (1 << 3)
	}

	if filter {
		v |= (1 << 2)
	}

	if store {
		v |= (1 << 1)
	}

	if relay {
		v |= (1 << 0)
	}

	return v
}

func GetENRandIP(addr ma.Multiaddr, wakuFlags WakuEnrBitfield, privK *ecdsa.PrivateKey) (*enode.Node, *net.TCPAddr, error) {
	ip, err := addr.ValueForProtocol(ma.P_IP4)
	if err != nil {
		return nil, nil, err
	}

	portStr, err := addr.ValueForProtocol(ma.P_TCP)
	if err != nil {
		return nil, nil, err
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, nil, err
	}

	tcpAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		return nil, nil, err
	}

	r := &enr.Record{}

	if port > 0 && port <= math.MaxUint16 {
		r.Set(enr.TCP(uint16(port))) // lgtm [go/incorrect-integer-conversion]
	} else {
		return nil, nil, fmt.Errorf("could not set port %d", port)
	}

	r.Set(enr.IP(net.ParseIP(ip)))
	r.Set(enr.WithEntry(WakuENRField, wakuFlags))

	err = enode.SignV4(r, privK)
	if err != nil {
		return nil, nil, err
	}

	node, err := enode.New(enode.ValidSchemes, r)

	return node, tcpAddr, err
}

func EnodeToMultiAddr(node *enode.Node) (ma.Multiaddr, error) {
	peerID, err := peer.IDFromPublicKey(&ECDSAPublicKey{node.Pubkey()})
	if err != nil {
		return nil, err
	}

	return ma.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d/p2p/%s", node.IP(), node.TCP(), peerID))
}

func EnodeToPeerInfo(node *enode.Node) (*peer.AddrInfo, error) {
	address, err := EnodeToMultiAddr(node)
	if err != nil {
		return nil, err
	}

	return peer.AddrInfoFromP2pAddr(address)
}
