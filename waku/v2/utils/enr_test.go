package utils

import (
	"encoding/binary"
	"errors"
	"math"
	"net"
	"testing"

	gcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestEnodeToMultiAddr(t *testing.T) {
	enr := "enr:-IS4QAmC_o1PMi5DbR4Bh4oHVyQunZblg4bTaottPtBodAhJZvxVlWW-4rXITPNg4mwJ8cW__D9FBDc9N4mdhyMqB-EBgmlkgnY0gmlwhIbRi9KJc2VjcDI1NmsxoQOevTdO6jvv3fRruxguKR-3Ge4bcFsLeAIWEDjrfaigNoN0Y3CCdl8"

	parsedNode := enode.MustParse(enr)
	expectedMultiAddr := "/ip4/134.209.139.210/tcp/30303/p2p/16Uiu2HAmPLe7Mzm8TsYUubgCAW1aJoeFScxrLj8ppHFivPo97bUZ"
	actualMultiAddr, err := enodeToMultiAddr(parsedNode)
	require.NoError(t, err)
	require.Equal(t, expectedMultiAddr, actualMultiAddr.String())
}

// TODO: this function is duplicated in localnode.go. Remove duplication
func updateLocalNode(localnode *enode.LocalNode, multiaddrs []ma.Multiaddr, ipAddr *net.TCPAddr, udpPort uint, wakuFlags WakuEnrBitfield, advertiseAddr *net.IP, shouldAutoUpdate bool, log *zap.Logger) error {
	localnode.SetFallbackUDP(int(udpPort))
	localnode.Set(enr.WithEntry(WakuENRField, wakuFlags))
	localnode.SetFallbackIP(net.IP{127, 0, 0, 1})

	if udpPort > math.MaxUint16 {
		return errors.New("invalid udp port number")
	}

	if advertiseAddr != nil {
		// An advertised address disables libp2p address updates
		// and discv5 predictions
		localnode.SetStaticIP(*advertiseAddr)
		localnode.Set(enr.TCP(uint16(ipAddr.Port))) // TODO: ipv6?
	} else if !shouldAutoUpdate {
		// We received a libp2p address update. Autoupdate is disabled
		// Using a static ip will disable endpoint prediction.
		localnode.SetStaticIP(ipAddr.IP)
		localnode.Set(enr.TCP(uint16(ipAddr.Port))) // TODO: ipv6?
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

		if ip6 != nil && !ip6.IsUnspecified() {
			localnode.Set(enr.IPv6(ip6))
			localnode.Set(enr.TCP6(ipAddr.Port))
		} else {
			localnode.Delete(enr.IPv6{})
			localnode.Delete(enr.TCP6(0))
		}
	}

	var addrAggr []ma.Multiaddr
	var err error
	for i := len(multiaddrs) - 1; i >= 0; i-- {
		addrAggr = append(addrAggr, multiaddrs[0:i]...)
		err = func() (err error) {
			defer func() {
				if e := recover(); e != nil {
					err = errors.New("could not write enr record")
				}
			}()

			var fieldRaw []byte
			for _, addr := range addrAggr {
				maRaw := addr.Bytes()
				maSize := make([]byte, 2)
				binary.BigEndian.PutUint16(maSize, uint16(len(maRaw)))

				fieldRaw = append(fieldRaw, maSize...)
				fieldRaw = append(fieldRaw, maRaw...)
			}

			if len(fieldRaw) != 0 {
				localnode.Set(enr.WithEntry(MultiaddrENRField, fieldRaw))
			}

			// This is to trigger the signing record err due to exceeding 300bytes limit
			_ = localnode.Node()

			return nil
		}()

		if err == nil {
			break
		}
	}

	// In case multiaddr could not be populated at all
	if err != nil {
		localnode.Delete(enr.WithEntry(MultiaddrENRField, struct{}{}))
	}

	return nil
}

func TestMultiaddr(t *testing.T) {

	key, _ := gcrypto.GenerateKey()
	wakuFlag := NewWakuEnrBitfield(true, true, true, true)

	//wss, _ := ma.NewMultiaddr("/dns4/www.somedomainname.com/tcp/443/wss")
	circuit1, _ := ma.NewMultiaddr("/dns4/node-02.gc-us-central1-a.status.prod.statusim.net/tcp/30303/p2p/16Uiu2HAmDQugwDHM3YeUp86iGjrUvbdw3JPRgikC7YoGBsT2ymMg/p2p-circuit")
	circuit2, _ := ma.NewMultiaddr("/dns4/node-01.gc-us-central1-a.status.prod.statusim.net/tcp/30303/p2p/16Uiu2HAmDQugwDHM3YeUp86iGjrUvbdw3JPRgikC7YoGBsT2ymMg/p2p-circuit")

	multiaddrValues := []ma.Multiaddr{
		//wss,
		circuit1,
		circuit2,
	}

	db, _ := enode.OpenDB("")
	localNode := enode.NewLocalNode(db, key)
	err := updateLocalNode(localNode, multiaddrValues, &net.TCPAddr{IP: net.IPv4(192, 168, 1, 241), Port: 60000}, 50000, wakuFlag, nil, false, Logger())
	require.NoError(t, err)

	_ = localNode.Node() // Should not panic

	_, err = Multiaddress(localNode.Node())
	require.NoError(t, err)
}
