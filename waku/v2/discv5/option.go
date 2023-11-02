package discv5

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/multiformats/go-multiaddr"
	"github.com/prometheus/client_golang/prometheus"
	cli "github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
	"github.com/waku-org/go-waku/waku/v2/dnsdisc"
	"go.uber.org/zap"
)

type DiscV5Parameters struct {
	// used by localNode
	AutoUpdate bool
	UdpPort    uint
	// for cli
	clinodes cli.StringSlice
	//
	enable        bool
	autoFindPeers bool
	bootnodes     []*enode.Node
	advertiseAddr []multiaddr.Multiaddr
	loopPredicate func(*enode.Node) bool
}

func (params *DiscV5Parameters) CliFlags() []cli.Flag {
	return []cli.Flag{
		altsrc.NewBoolFlag(&cli.BoolFlag{
			Name:        "discv5-discovery",
			Usage:       "Enable discovering nodes via Node Discovery v5",
			Destination: &params.enable,
			EnvVars:     []string{"WAKUNODE2_DISCV5_DISCOVERY"},
		}), altsrc.NewStringSliceFlag(&cli.StringSliceFlag{
			Name:        "discv5-bootstrap-node",
			Usage:       "Text-encoded ENR for bootstrap node. Used when connecting to the network. Option may be repeated",
			Destination: &params.clinodes,
			EnvVars:     []string{"WAKUNODE2_DISCV5_BOOTSTRAP_NODE"},
		}), altsrc.NewUintFlag(&cli.UintFlag{
			Name:        "discv5-udp-port",
			Value:       9000,
			Usage:       "Listening UDP port for Node Discovery v5.",
			Destination: &params.UdpPort,
			EnvVars:     []string{"WAKUNODE2_DISCV5_UDP_PORT"},
		}), altsrc.NewBoolFlag(&cli.BoolFlag{
			Name:        "discv5-enr-auto-update",
			Usage:       "Discovery can automatically update its ENR with the IP address as seen by other nodes it communicates with.",
			Destination: &params.AutoUpdate,
			EnvVars:     []string{"WAKUNODE2_DISCV5_ENR_AUTO_UPDATE"},
		}),
	}
}

func GetDiscv5Params(udpPort uint, bootnodes []*enode.Node, autoUpdate bool) DiscV5Parameters {
	return DiscV5Parameters{
		enable:     true,
		UdpPort:    udpPort,
		bootnodes:  bootnodes,
		AutoUpdate: autoUpdate,
	}
}
func WithAutoUpdate(autoUpdate bool) DiscoveryV5Option {
	return func(params *DiscV5Parameters) {
		params.AutoUpdate = autoUpdate
	}
}

// WithBootnodes is an option used to specify the bootstrap nodes to use with DiscV5
func WithBootnodes(bootnodes []*enode.Node) DiscoveryV5Option {
	return func(params *DiscV5Parameters) {
		params.bootnodes = bootnodes
	}
}

func WithAdvertiseAddr(addr []multiaddr.Multiaddr) DiscoveryV5Option {
	return func(params *DiscV5Parameters) {
		params.advertiseAddr = addr
	}
}

func WithUDPPort(port uint) DiscoveryV5Option {
	return func(params *DiscV5Parameters) {
		params.UdpPort = port
	}
}

func WithPredicate(predicate func(*enode.Node) bool) DiscoveryV5Option {
	return func(params *DiscV5Parameters) {
		params.loopPredicate = predicate
	}
}

func WithAutoFindPeers(find bool) DiscoveryV5Option {
	return func(params *DiscV5Parameters) {
		params.autoFindPeers = find
	}
}

func cliListToENode(logger *zap.Logger, slice cli.StringSlice) []*enode.Node {
	var bootnodes []*enode.Node
	for _, addr := range slice.Value() {
		bootnode, err := enode.Parse(enode.ValidSchemes, addr)
		if err != nil {
			logger.Fatal("parsing ENR", zap.Error(err))
		}
		bootnodes = append(bootnodes, bootnode)
	}
	return bootnodes
}

func (params *DiscV5Parameters) Service(priv *ecdsa.PrivateKey, localnode *enode.LocalNode, peerConnector PeerConnector, reg prometheus.Registerer, log *zap.Logger, discoveredNodes []dnsdisc.DiscoveredNode, advertiseAddrs []multiaddr.Multiaddr) (*DiscoveryV5, error) {
	if !params.enable {
		return nil, nil
	}
	bootnodes := cliListToENode(log, params.clinodes)
	for _, n := range discoveredNodes {
		if n.ENR != nil {
			bootnodes = append(bootnodes, n.ENR)
		}
	}
	discV5Options := []DiscoveryV5Option{
		WithBootnodes(params.bootnodes),
		WithUDPPort(params.UdpPort),
		WithAutoUpdate(params.AutoUpdate),
		WithAdvertiseAddr(advertiseAddrs),
		WithBootnodes(bootnodes),
	}

	return NewDiscoveryV5(priv, localnode, peerConnector, reg, log, discV5Options...)
}
