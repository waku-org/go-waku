package waku

type RendezvousOptions struct {
	Enable bool     `long:"rendezvous" description:"Enable rendezvous protocol for peer discovery"`
	Nodes  []string `long:"rendezvous-nodes" description:"Multiaddrs of waku2 rendezvous nodes. Argument may be repeated"`
}
type RendezvousServerOptions struct {
	Enable bool   `long:"rendezvous-server" description:"Node will act as rendezvous server"`
	DBPath string `long:"rendezvous-db-path" description:"Path where peer records database will be stored" default:"/tmp/rendezvous"`
}

type RelayOptions struct {
	Disable      bool     `long:"no-relay" description:"Disable relay protocol"`
	Topics       []string `long:"topics" description:"List of topics to listen"`
	PeerExchange bool     `long:"peer-exchange" description:"Enable GossipSub Peer Exchange"`
}

type FilterOptions struct {
	Enable bool     `long:"filter" description:"Enable filter protocol"`
	Nodes  []string `long:"filter-nodes" description:"Multiaddr of peers to request content filtering of messages. Argument may be repeated"`
}

type LightpushOptions struct {
	Enable bool     `long:"lightpush" description:"Enable lightpush protocol"`
	Nodes  []string `long:"lightpush-nodes" description:"Multiaddr of peers to request lightpush of published messages. Argument may be repeated"`
}

type StoreOptions struct {
	Enable bool     `long:"store" description:"Enable store protocol"`
	Nodes  []string `long:"store-nodes" description:"Multiaddr of peers to request stored messages. Argument may be repeated"`
}

type DNSDiscoveryOptions struct {
	Enable     bool   `long:"dns-discovery" description:"Enable DNS discovery"`
	URL        string `long:"dns-discovery-url" description:"URL for DNS node list in format 'enrtree://<key>@<fqdn>'"`
	Nameserver string `long:"dns-discovery-nameserver" description:"DNS nameserver IP to query (empty to use system's default)"`
}

type MetricsOptions struct {
	Enable  bool   `long:"metrics" description:"Enable the metrics server"`
	Address string `long:"metrics-address" description:"Listening address of the metrics server" default:"127.0.0.1"`
	Port    int    `long:"metrics-port" description:"Listening HTTP port of the metrics server" default:"8008"`
}

type Options struct {
	// Example of optional value
	Port        int      `short:"p" long:"port" description:"Libp2p TCP listening port (0 for random)" default:"9000"`
	EnableWS    bool     `long:"ws" description:"Enable websockets support"`
	WSPort      int      `long:"ws-port" description:"Libp2p TCP listening port for websocket connection (0 for random)" default:"9001"`
	NodeKey     string   `long:"nodekey" description:"P2P node private key as hex. Can also be set with GOWAKU-NODEKEY env variable (default random)"`
	KeyFile     string   `long:"key-file" description:"Path to a file containing the private key for the P2P node" default:"./nodekey"`
	GenerateKey bool     `long:"generate-key" description:"Generate private key file at path specified in --key-file"`
	Overwrite   bool     `long:"overwrite" description:"When generating a keyfile, overwrite the nodekey file if it already exists"`
	StaticNodes []string `long:"staticnodes" description:"Multiaddr of peer to directly connect with. Argument may be repeated"`
	KeepAlive   int      `long:"keep-alive" default:"20" description:"Interval in seconds for pinging peers to keep the connection alive."`
	UseDB       bool     `long:"use-db" description:"Use SQLiteDB to persist information"`
	DBPath      string   `long:"dbpath" default:"./store.db" description:"Path to DB file"`

	Relay            RelayOptions            `group:"Relay Options"`
	Store            StoreOptions            `group:"Store Options"`
	Filter           FilterOptions           `group:"Filter Options"`
	LightPush        LightpushOptions        `group:"LightPush Options"`
	Rendezvous       RendezvousOptions       `group:"Rendezvous Options"`
	RendezvousServer RendezvousServerOptions `group:"Rendezvous Server Options"`
	DNSDiscovery     DNSDiscoveryOptions     `group:"DNS Discovery Options"`
	Metrics          MetricsOptions          `group:"Metrics Options"`
}
