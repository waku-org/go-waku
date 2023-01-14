package utils

import (
	"crypto/ecdsa"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"net"
	"strconv"

	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"go.uber.org/zap"
)

// WakuENRField is the name of the ENR field that contains information about which protocols are supported by the node
const WakuENRField = "waku2"

// MultiaddrENRField is the name of the ENR field that will contain multiaddresses that cannot be described using the
// already available ENR fields (i.e. in the case of websocket connections)
const MultiaddrENRField = "multiaddrs"

// P2PCircuit is a field that indicates the public key of a circuit relay node
const P2PCircuit = "p2p-circuit"

// WakuEnrBitfield is a8-bit flag field to indicate Waku capabilities. Only the 4 LSBs are currently defined according to RFC31 (https://rfc.vac.dev/spec/31/).
type WakuEnrBitfield = uint8

// NewWakuEnrBitfield creates a WakuEnrBitField whose value will depend on which protocols are enabled in the node
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

func IsCircuitRelayAddress(m multiaddr.Multiaddr) bool {
	_, err := m.ValueForProtocol(multiaddr.P_CIRCUIT)
	return err == nil
}

// GetENRandIP returns a enr Node and TCP address obtained from a multiaddress. priv key and protocols supported
func GetENRandIP(addr multiaddr.Multiaddr, wakuFlags WakuEnrBitfield, privK *ecdsa.PrivateKey) (*enode.Node, *net.TCPAddr, error) {

	isRelay := IsCircuitRelayAddress(addr)
	var p2pCircuitAddr multiaddr.Multiaddr
	if isRelay {
		addr, p2pCircuitAddr = multiaddr.SplitFunc(addr, func(c multiaddr.Component) bool {
			return c.Protocol().Code == multiaddr.P_CIRCUIT
		})
	}

	var ip string
	dns4, err := addr.ValueForProtocol(multiaddr.P_DNS4)
	if err != nil {
		ip, err = addr.ValueForProtocol(multiaddr.P_IP4)
		if err != nil {
			return nil, nil, err
		}
	} else {
		netIP, err := net.ResolveIPAddr("ip4", dns4)
		if err != nil {
			return nil, nil, err
		}
		ip = netIP.String()
	}

	portStr, err := addr.ValueForProtocol(multiaddr.P_TCP)
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

	var relayP2PAddr multiaddr.Multiaddr
	var relayCompressedPublicKey []byte
	var nodeP2PAddr multiaddr.Multiaddr
	if isRelay {
		relayP2PAddr, relayCompressedPublicKey, err = P2PAddr(addr)
		if err != nil {
			return nil, nil, err
		}

		nodeP2PAddr, _, err = P2PAddr(p2pCircuitAddr)
		if err != nil {
			return nil, nil, err
		}
	} else {
		nodeP2PAddr, _, err = P2PAddr(addr)
		if err != nil {
			return nil, nil, err
		}
	}

	var multiaddrItems []multiaddr.Multiaddr

	// 31/WAKU2-ENR

	_, err = addr.ValueForProtocol(multiaddr.P_WS)
	if err == nil {
		multiaddrItems = append(multiaddrItems, addr)
	}

	_, err = addr.ValueForProtocol(multiaddr.P_WSS)
	if err == nil {
		multiaddrItems = append(multiaddrItems, addr)
	}

	var fieldRaw []byte
	for _, ma := range multiaddrItems {
		var maRaw []byte
		if isRelay {
			maRaw = ma.Decapsulate(relayP2PAddr).Bytes()
		} else {
			maRaw = ma.Decapsulate(nodeP2PAddr).Bytes()
		}

		maSize := make([]byte, 2)
		binary.BigEndian.PutUint16(maSize, uint16(len(maRaw)))

		fieldRaw = append(fieldRaw, maSize...)
		fieldRaw = append(fieldRaw, maRaw...)
	}

	if len(fieldRaw) != 0 {
		r.Set(enr.WithEntry(MultiaddrENRField, fieldRaw))
	}

	if isRelay {
		r.Set(enr.WithEntry(P2PCircuit, relayCompressedPublicKey))
	}

	r.Set(enr.IP(net.ParseIP(ip)))
	r.Set(enr.WithEntry(WakuENRField, wakuFlags))

	err = enode.SignV4(r, privK)
	if err != nil {
		return nil, nil, err
	}

	node, err := enode.New(enode.ValidSchemes, r)
	fmt.Println(node)
	return node, tcpAddr, err
}

func P2PAddr(addr multiaddr.Multiaddr) (multiaddr.Multiaddr, []byte, error) {
	p2pVal, err := addr.ValueForProtocol(multiaddr.P_P2P)
	if err != nil {
		return nil, nil, err
	}

	pid, err := peer.Decode(p2pVal)
	if err != nil {
		return nil, nil, err
	}

	pubkey, err := pid.ExtractPublicKey()
	if err != nil {
		return nil, nil, err
	}

	compressedPubkey, err := pubkey.(*crypto.Secp256k1PublicKey).Raw()
	if err != nil {
		return nil, nil, err
	}

	m, err := multiaddr.NewMultiaddr("/p2p/" + p2pVal)
	if err != nil {
		return nil, nil, err
	}

	return m, compressedPubkey, nil
}

func bytesToPeerID(peerIDBytes []byte) (peer.ID, error) {
	x, y := secp256k1.DecompressPubkey(peerIDBytes)
	if x == nil {
		return "", fmt.Errorf("invalid public key")
	}
	relayPubKey := EcdsaPubKeyToSecp256k1PublicKey(&ecdsa.PublicKey{X: x, Y: y, Curve: secp256k1.S256()})
	return peer.IDFromPublicKey(relayPubKey)
}

// EnodeToMultiaddress converts an enode without `multiaddr` key into a multiaddress
func enodeToMultiAddr(node *enode.Node) (multiaddr.Multiaddr, error) {
	pubKey := EcdsaPubKeyToSecp256k1PublicKey(node.Pubkey())
	peerID, err := peer.IDFromPublicKey(pubKey)
	if err != nil {
		return nil, err
	}

	var relayPeerIDBytes []byte
	if err := node.Record().Load(enr.WithEntry(P2PCircuit, &relayPeerIDBytes)); err == nil {
		relayPeerID, err := bytesToPeerID(relayPeerIDBytes)
		if err != nil {
			return nil, err
		}
		return multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d/p2p/%s/p2p-circuit/p2p/%s", node.IP(), node.TCP(), relayPeerID, peerID))
	} else if !enr.IsNotFound(err) {
		// Something went wrong loading the ENR
		return nil, err
	} else {
		return multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d/p2p/%s", node.IP(), node.TCP(), peerID))
	}
}

// Multiaddress is used to extract all the multiaddresses that are part of a ENR record
func Multiaddress(node *enode.Node) ([]multiaddr.Multiaddr, error) {
	pubKey := EcdsaPubKeyToSecp256k1PublicKey(node.Pubkey())
	peerID, err := peer.IDFromPublicKey(pubKey)
	if err != nil {
		return nil, err
	}

	var multiaddrRaw []byte
	if err := node.Record().Load(enr.WithEntry(MultiaddrENRField, &multiaddrRaw)); err != nil {
		if enr.IsNotFound(err) {
			Logger().Debug("trying to convert enode to multiaddress, since I could not retrieve multiaddress field for node ", zap.Any("enode", node))
			addr, err := enodeToMultiAddr(node)
			if err != nil {
				return nil, err
			}
			return []multiaddr.Multiaddr{addr}, nil
		}
		return nil, err
	} else {

		if len(multiaddrRaw) < 2 {
			return nil, errors.New("invalid multiaddress field length")
		}

		var relayPeerIDBytes []byte
		var isRelay = false
		var nodeAddr multiaddr.Multiaddr
		var p2pCircuitAddr multiaddr.Multiaddr
		if err := node.Record().Load(enr.WithEntry(P2PCircuit, &relayPeerIDBytes)); err == nil {
			isRelay = true
			relayPeerID, err := bytesToPeerID(relayPeerIDBytes)
			if err != nil {
				return nil, err
			}
			nodeAddr, err = multiaddr.NewMultiaddr(fmt.Sprintf("/p2p/%s", relayPeerID.Pretty()))
			if err != nil {
				return nil, err
			}
			p2pCircuitAddr, err = multiaddr.NewMultiaddr(fmt.Sprintf("/p2p-circuit/p2p/%s", peerID.Pretty()))
			if err != nil {
				return nil, err
			}
		} else if !enr.IsNotFound(err) {
			return nil, err
		} else {
			// No circuit relay
			nodeAddr, err = multiaddr.NewMultiaddr(fmt.Sprintf("/p2p/%s", peerID.Pretty()))
			if err != nil {
				return nil, err
			}
		}

		var result []multiaddr.Multiaddr
		offset := 0
		for {
			maSize := binary.BigEndian.Uint16(multiaddrRaw[offset : offset+2])
			if len(multiaddrRaw) < offset+2+int(maSize) {
				return nil, errors.New("invalid multiaddress field length")
			}
			maRaw := multiaddrRaw[offset+2 : offset+2+int(maSize)]
			addr, err := multiaddr.NewMultiaddrBytes(maRaw)
			if err != nil {
				return nil, fmt.Errorf("invalid multiaddress field length")
			}

			if isRelay {
				result = append(result, addr.Encapsulate(nodeAddr).Encapsulate(p2pCircuitAddr))
			} else {
				result = append(result, addr.Encapsulate(nodeAddr))
			}

			offset += 2 + int(maSize)
			if offset >= len(multiaddrRaw) {
				break
			}
		}

		return result, nil
	}
}

// EnodeToPeerInfo extracts the peer ID and multiaddresses defined in an ENR
func EnodeToPeerInfo(node *enode.Node) (*peer.AddrInfo, error) {
	addresses, err := Multiaddress(node)
	if err != nil {
		return nil, err
	}

	res, err := peer.AddrInfosFromP2pAddrs(addresses...)
	if err != nil {
		return nil, err
	}

	return &res[0], nil
}
