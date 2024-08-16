package enr

import (
	"net"
	"testing"

	gcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

func TestEnodeToMultiAddr(t *testing.T) {
	enr := "enr:-QESuEC1p_s3xJzAC_XlOuuNrhVUETmfhbm1wxRGis0f7DlqGSw2FM-p2Ugl_r25UHQJ3f1rIRrpzxJXSMaJe4yk1XFSAYJpZIJ2NIJpcISygI2rim11bHRpYWRkcnO4XAArNiZub2RlLTAxLmRvLWFtczMud2FrdS50ZXN0LnN0YXR1c2ltLm5ldAZ2XwAtNiZub2RlLTAxLmRvLWFtczMud2FrdS50ZXN0LnN0YXR1c2ltLm5ldAYfQN4DgnJzkwABCAAAAAEAAgADAAQABQAGAAeJc2VjcDI1NmsxoQJATXRSRSUyTw_QLB6H_U3oziVQgNRgrXpK7wp2AMyNxYN0Y3CCdl-DdWRwgiMohXdha3UyDw"

	parsedNode := enode.MustParse(enr)
	expectedMultiAddr := "/ip4/178.128.141.171/tcp/30303/p2p/16Uiu2HAkykgaECHswi3YKJ5dMLbq2kPVCo89fcyTd38UcQD6ej5W"
	actualMultiAddr, err := enodeToMultiAddr(parsedNode)
	require.NoError(t, err)
	require.Equal(t, expectedMultiAddr, actualMultiAddr.String())
}

// TODO: this function is duplicated in localnode.go. Remove duplication

func updateLocalNode(localnode *enode.LocalNode, multiaddrs []ma.Multiaddr, ipAddr *net.TCPAddr, udpPort uint, wakuFlags WakuEnrBitfield, advertiseAddr *net.IP, shouldAutoUpdate bool) error {
	var options []ENROption
	options = append(options, WithUDPPort(udpPort))
	options = append(options, WithWakuBitfield(wakuFlags))
	options = append(options, WithMultiaddress(multiaddrs...))

	if advertiseAddr != nil {
		// An advertised address disables libp2p address updates
		// and discv5 predictions
		nip := &net.TCPAddr{
			IP:   *advertiseAddr,
			Port: ipAddr.Port,
		}
		options = append(options, WithIP(nip))
	} else if !shouldAutoUpdate {
		// We received a libp2p address update. Autoupdate is disabled
		// Using a static ip will disable endpoint prediction.
		options = append(options, WithIP(ipAddr))
	} else {
		// We received a libp2p address update, but we should still
		// allow discv5 to update the enr record. We set the localnode
		// keys manually. It's possible that the ENR record might get
		// updated automatically
		ip4 := ipAddr.IP.To4()
		ip6 := ipAddr.IP.To16()
		if ip4 != nil && !ip4.IsUnspecified() {
			localnode.Set(enr.IPv4(ip4))
			localnode.Set(enr.TCP(uint16(ipAddr.Port)))
		} else {
			localnode.Delete(enr.IPv4{})
			localnode.Delete(enr.TCP(0))
		}

		if ip4 == nil && ip6 != nil && !ip6.IsUnspecified() {
			localnode.Set(enr.IPv6(ip6))
			localnode.Set(enr.TCP6(ipAddr.Port))
		} else {
			localnode.Delete(enr.IPv6{})
			localnode.Delete(enr.TCP6(0))
		}
	}

	return Update(utils.Logger(), localnode, options...)
}

func TestMultiaddr(t *testing.T) {

	key, _ := gcrypto.GenerateKey()
	wakuFlag := NewWakuEnrBitfield(true, true, true, true)

	//wss, _ := ma.NewMultiaddr("/dns4/www.somedomainname.com/tcp/443/wss")
	circuit1, _ := ma.NewMultiaddr("/dns4/node-02.gc-us-central1-a.status.prod.status.im/tcp/30303/p2p/16Uiu2HAmDQugwDHM3YeUp86iGjrUvbdw3JPRgikC7YoGBsT2ymMg/p2p-circuit")
	circuit2, _ := ma.NewMultiaddr("/dns4/node-01.gc-us-central1-a.status.prod.status.im/tcp/30303/p2p/16Uiu2HAmDQugwDHM3YeUp86iGjrUvbdw3JPRgikC7YoGBsT2ymMg/p2p-circuit")

	multiaddrValues := []ma.Multiaddr{
		//wss,
		circuit1,
		circuit2,
	}

	db, _ := enode.OpenDB("")
	localNode := enode.NewLocalNode(db, key)
	err := updateLocalNode(localNode, multiaddrValues, &net.TCPAddr{IP: net.IPv4(192, 168, 1, 241), Port: 60000}, 50000, wakuFlag, nil, false)
	require.NoError(t, err)

	_ = localNode.Node() // Should not panic

	_, _, err = Multiaddress(localNode.Node())
	require.NoError(t, err)
}
