package waku

import "time"

type RendezvousOptions struct {
	Enable bool     `long:"rendezvous" description:"Enable rendezvous protocol for peer discovery"`
	Nodes  []string `long:"rendezvous-node" description:"Multiaddr of a waku2 rendezvous node. Option may be repeated"`
}

type RendezvousServerOptions struct {
	Enable bool   `long:"rendezvous-server" description:"Node will act as rendezvous server"`
	DBPath string `long:"rendezvous-db-path" description:"Path where peer records database will be stored" default:"/tmp/rendezvous"`
}

type DiscV5Options struct {
	Enable     bool     `long:"discv5-discovery" description:"Enable discovering nodes via Node Discovery v5"`
	Nodes      []string `long:"discv5-bootstrap-node" description:"Text-encoded ENR for bootstrap node. Used when connecting to the network. Option may be repeated"`
	Port       int      `long:"discv5-udp-port" description:"Listening UDP port for Node Discovery v5." default:"9000"`
	AutoUpdate bool     `long:"discv5-enr-auto-update" description:"Discovery can automatically update its ENR with the IP address as seen by other nodes it communicates with." `
}

type RelayOptions struct {
	Disable                bool     `long:"no-relay" description:"Disable relay protocol"`
	Topics                 []string `long:"topics" description:"List of topics to listen"`
	PeerExchange           bool     `long:"peer-exchange" description:"Enable GossipSub Peer Exchange"`
	MinRelayPeersToPublish int      `long:"min-relay-peers-to-publish" description:"Minimum number of peers to publish to Relay" default:"1"`
}

type FilterOptions struct {
	Enable          bool     `long:"filter" description:"Enable filter protocol"`
	DisableFullNode bool     `long:"light-client" description:"Don't accept filter subscribers"`
	Nodes           []string `long:"filter-node" description:"Multiaddr of a peer that supports filter protocol. Option may be repeated"`
}

// LightpushOptions are settings used to enable the lightpush protocol. This is
// a lightweight protocol used to avoid having to run the relay protocol which
// is more resource intensive. With this protocol a message is pushed to a peer
// that supports both the lightpush protocol and relay protocol. That peer will
// broadcast the message and return a confirmation that the message was
// broadcasted
type LightpushOptions struct {
	Enable bool     `long:"lightpush" description:"Enable lightpush protocol"`
	Nodes  []string `long:"lightpush-node" description:"Multiaddr of a peer that supports lightpush protocol. Option may be repeated"`
}

// StoreOptions are settings used for enabling the store protocol, used to
// retrieve message history from other nodes as well as acting as a store
// node and provide message history to nodes that ask for it.
type StoreOptions struct {
	Enable               bool     `long:"store" description:"Enable store protocol"`
	ShouldResume         bool     `long:"resume" description:"fix the gap in message history"`
	RetentionMaxDays     int      `long:"keep-history-days" description:"maximum number of days before a message is removed from the store" default:"30"`
	RetentionMaxMessages int      `long:"max-history-messages" description:"maximum number of messages to store" default:"50000"`
	Nodes                []string `long:"store-node" description:"Multiaddr of a peer that supports store protocol. Option may be repeated"`
}

// SwapOptions are settings used for configuring the swap protocol
type SwapOptions struct {
	Mode                int `long:"swap-mode" description:"Swap mode: 0=soft, 1=mock, 2=hard" default:"0"`
	PaymentThreshold    int `long:"swap-payment-threshold" description:"Threshold for payment" default:"100"`
	DisconnectThreshold int `long:"swap-disconnect-threshold" description:"Threshold for disconnecting" default:"-100"`
}

func (s *StoreOptions) RetentionMaxDaysDuration() time.Duration {
	return time.Duration(s.RetentionMaxDays) * time.Hour * 24
}

// DNSDiscoveryOptions are settings used for enabling DNS-based discovery
// protocol that stores merkle trees in DNS records which contain connection
// information for nodes. It's very useful for bootstrapping a p2p network.
type DNSDiscoveryOptions struct {
	Enable     bool   `long:"dns-discovery" description:"Enable DNS discovery"`
	URL        string `long:"dns-discovery-url" description:"URL for DNS node list in format 'enrtree://<key>@<fqdn>'"`
	Nameserver string `long:"dns-discovery-nameserver" description:"DNS nameserver IP to query (empty to use system's default)"`
}

// MetricsOptions are settings used to start a prometheus server for obtaining
// useful node metrics to monitor the health of behavior of the go-waku node.
type MetricsOptions struct {
	Enable  bool   `long:"metrics" description:"Enable the metrics server"`
	Address string `long:"metrics-address" description:"Listening address of the metrics server" default:"127.0.0.1"`
	Port    int    `long:"metrics-port" description:"Listening HTTP port of the metrics server" default:"8008"`
}

// RPCServerOptions are settings used to start a json rpc server
type RPCServerOptions struct {
	Enable  bool   `long:"rpc" description:"Enable the rpc server"`
	Port    int    `long:"rpc-port" description:"Listening port of the rpc server" default:"8009"`
	Address string `long:"rpc-address" description:"Listening address of the rpc server" default:"127.0.0.1"`
}

// Options contains all the available features and settings that can be
// configured via flags when executing go-waku as a service.
type Options struct {
	Port             int      `short:"p" long:"port" description:"Libp2p TCP listening port (0 for random)" default:"60000"`
	Address          string   `long:"address" description:"Listening address" default:"0.0.0.0"`
	EnableWS         bool     `long:"ws" description:"Enable websockets support"`
	WSPort           int      `long:"ws-port" description:"Libp2p TCP listening port for websocket connection (0 for random)" default:"60001"`
	WSAddress        string   `long:"ws-address" description:"Listening address for websocket connections" default:"0.0.0.0"`
	NodeKey          string   `long:"nodekey" description:"P2P node private key as hex. Can also be set with GOWAKU-NODEKEY env variable (default random)"`
	KeyFile          string   `long:"key-file" description:"Path to a file containing the private key for the P2P node" default:"./nodekey"`
	GenerateKey      bool     `long:"generate-key" description:"Generate private key file at path specified in --key-file"`
	Overwrite        bool     `long:"overwrite" description:"When generating a keyfile, overwrite the nodekey file if it already exists"`
	StaticNodes      []string `long:"static-node" description:"Multiaddr of peer to directly connect with. Option may be repeated"`
	KeepAlive        int      `long:"keep-alive" default:"20" description:"Interval in seconds for pinging peers to keep the connection alive."`
	UseDB            bool     `long:"use-db" description:"Use SQLiteDB to persist information"`
	DBPath           string   `long:"dbpath" default:"./store.db" description:"Path to DB file"`
	AdvertiseAddress string   `long:"advertise-address" default:"" description:"External address to advertise to other nodes (overrides --address and --ws-address flags)"`
	ShowAddresses    bool     `long:"show-addresses" description:"Display listening addresses according to current configuration"`
	LogLevel         string   `short:"l" long:"log-level" description:"Define the logging level, supported strings are: DEBUG, INFO, WARN, ERROR, DPANIC, PANIC, FATAL, and their lower-case forms." default:"INFO"`

	Relay            RelayOptions            `group:"Relay Options"`
	Store            StoreOptions            `group:"Store Options"`
	Swap             SwapOptions             `group:"Swap Options"`
	Filter           FilterOptions           `group:"Filter Options"`
	LightPush        LightpushOptions        `group:"LightPush Options"`
	DiscV5           DiscV5Options           `group:"DiscoveryV5 Options"`
	Rendezvous       RendezvousOptions       `group:"Rendezvous Options"`
	RendezvousServer RendezvousServerOptions `group:"Rendezvous Server Options"`
	DNSDiscovery     DNSDiscoveryOptions     `group:"DNS Discovery Options"`
	Metrics          MetricsOptions          `group:"Metrics Options"`
	RPCServer        RPCServerOptions        `group:"RPC Server Options"`
}
