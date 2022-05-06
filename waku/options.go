package waku

import (
	"time"

	"github.com/urfave/cli/v2"
)

type RendezvousOptions struct {
	Enable bool
	Nodes  cli.StringSlice
}

type RendezvousServerOptions struct {
	Enable bool
	DBPath string
}

type DiscV5Options struct {
	Enable     bool
	Nodes      cli.StringSlice
	Port       int
	AutoUpdate bool
}

type RelayOptions struct {
	Enable                 bool
	Topics                 cli.StringSlice
	PeerExchange           bool
	MinRelayPeersToPublish int
}

type FilterOptions struct {
	Enable          bool
	DisableFullNode bool
	Nodes           cli.StringSlice
	Timeout         int
}

// LightpushOptions are settings used to enable the lightpush protocol. This is
// a lightweight protocol used to avoid having to run the relay protocol which
// is more resource intensive. With this protocol a message is pushed to a peer
// that supports both the lightpush protocol and relay protocol. That peer will
// broadcast the message and return a confirmation that the message was
// broadcasted
type LightpushOptions struct {
	Enable bool
	Nodes  cli.StringSlice
}

// StoreOptions are settings used for enabling the store protocol, used to
// retrieve message history from other nodes as well as acting as a store
// node and provide message history to nodes that ask for it.
type StoreOptions struct {
	Enable               bool
	ShouldResume         bool
	RetentionMaxDays     int
	RetentionMaxMessages int
	Nodes                cli.StringSlice
}

// SwapOptions are settings used for configuring the swap protocol
type SwapOptions struct {
	Mode                int
	PaymentThreshold    int
	DisconnectThreshold int
}

func (s *StoreOptions) RetentionMaxDaysDuration() time.Duration {
	return time.Duration(s.RetentionMaxDays) * time.Hour * 24
}

// DNSDiscoveryOptions are settings used for enabling DNS-based discovery
// protocol that stores merkle trees in DNS records which contain connection
// information for nodes. It's very useful for bootstrapping a p2p network.
type DNSDiscoveryOptions struct {
	Enable     bool
	URL        string
	Nameserver string
}

// MetricsOptions are settings used to start a prometheus server for obtaining
// useful node metrics to monitor the health of behavior of the go-waku node.
type MetricsOptions struct {
	Enable  bool
	Address string
	Port    int
}

// RPCServerOptions are settings used to start a json rpc server
type RPCServerOptions struct {
	Enable  bool
	Port    int
	Address string
	Admin   bool
	Private bool
}

type WSOptions struct {
	Enable   bool
	Port     int
	Address  string
	Secure   bool
	KeyPath  string
	CertPath string
}

// Options contains all the available features and settings that can be
// configured via flags when executing go-waku as a service.
type Options struct {
	Port             int
	Address          string
	NodeKey          string
	KeyFile          string
	GenerateKey      bool
	Overwrite        bool
	StaticNodes      cli.StringSlice
	KeepAlive        int
	UseDB            bool
	DBPath           string
	AdvertiseAddress string
	ShowAddresses    bool
	LogLevel         string

	Websocket        WSOptions
	Relay            RelayOptions
	Store            StoreOptions
	Swap             SwapOptions
	Filter           FilterOptions
	LightPush        LightpushOptions
	DiscV5           DiscV5Options
	Rendezvous       RendezvousOptions
	RendezvousServer RendezvousServerOptions
	DNSDiscovery     DNSDiscoveryOptions
	Metrics          MetricsOptions
	RPCServer        RPCServerOptions
}
