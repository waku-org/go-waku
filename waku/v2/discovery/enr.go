package discovery

import (
	"fmt"

	"github.com/ethereum/go-ethereum/p2p/dnsdisc"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/libp2p/go-libp2p-core/peer"

	ma "github.com/multiformats/go-multiaddr"
)

// RetrieveNodes returns a list of multiaddress given a url to a DNS discoverable
// ENR tree
func RetrieveNodes(url string) ([]ma.Multiaddr, error) {
	var multiAddrs []ma.Multiaddr

	client := dnsdisc.NewClient(dnsdisc.Config{})

	tree, err := client.SyncTree(url)
	if err != nil {
		return nil, err
	}

	for _, node := range tree.Nodes() {
		m, err := enodeToMultiAddr(node)
		if err != nil {
			return nil, err
		}

		multiAddrs = append(multiAddrs, m)

	}

	return multiAddrs, nil
}

func enodeToMultiAddr(node *enode.Node) (ma.Multiaddr, error) {
	peerID, err := peer.IDFromPublicKey(&ECDSAPublicKey{node.Pubkey()})
	if err != nil {
		return nil, err
	}

	return ma.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d/p2p/%s", node.IP(), node.TCP(), peerID))
}
