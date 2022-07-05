package main

import (
	"os"

	logging "github.com/ipfs/go-log"
	"github.com/status-im/go-waku/waku"
	"github.com/status-im/go-waku/waku/v2/utils"
	"github.com/urfave/cli/v2"
)

var options waku.Options

func main() {
	app := &cli.App{
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:        "tcp-port",
				Aliases:     []string{"port", "p"},
				Value:       60000,
				Usage:       "Libp2p TCP listening port (0 for random)",
				Destination: &options.Port,
			},
			&cli.StringFlag{
				Name:        "address",
				Aliases:     []string{"host", "listen-address"},
				Value:       "0.0.0.0",
				Usage:       "Listening address",
				Destination: &options.Address,
			},
			&cli.BoolFlag{
				Name:        "websocket-support",
				Aliases:     []string{"ws"},
				Usage:       "Enable websockets support",
				Destination: &options.Websocket.Enable,
			},
			&cli.IntFlag{
				Name:        "websocket-port",
				Aliases:     []string{"ws-port"},
				Value:       60001,
				Usage:       "Libp2p TCP listening port for websocket connection (0 for random)",
				Destination: &options.Websocket.Port,
			},
			&cli.StringFlag{
				Name:        "websocket-address",
				Aliases:     []string{"ws-address"},
				Value:       "0.0.0.0",
				Usage:       "Listening address for websocket connections",
				Destination: &options.Websocket.Address,
			},
			&cli.BoolFlag{
				Name:        "websocket-secure-support",
				Aliases:     []string{"wss"},
				Usage:       "Enable secure websockets support",
				Destination: &options.Websocket.Secure,
			},
			&cli.StringFlag{
				Name:        "websocket-secure-key-path",
				Aliases:     []string{"wss-key"},
				Value:       "/path/to/key.txt",
				Usage:       "Secure websocket key path",
				Destination: &options.Websocket.KeyPath,
			},
			&cli.StringFlag{
				Name:        "websocket-secure-cert-path",
				Aliases:     []string{"wss-cert"},
				Value:       "/path/to/cert.txt",
				Usage:       "Secure websocket certificate path",
				Destination: &options.Websocket.CertPath,
			},
			&cli.StringFlag{
				Name:        "dns4-domain-name",
				Value:       "",
				Usage:       "The domain name resolving to the node's public IPv4 address",
				Destination: &options.Dns4DomainName,
			},
			&cli.StringFlag{
				Name:        "nodekey",
				Usage:       "P2P node private key as hex. Can also be set with GOWAKU-NODEKEY env variable (default random)",
				Destination: &options.NodeKey,
			},
			&cli.StringFlag{
				Name:        "key-file",
				Value:       "./nodekey",
				Usage:       "Path to a file containing the private key for the P2P node",
				Destination: &options.KeyFile,
			},
			&cli.BoolFlag{
				Name:        "generate-key",
				Usage:       "Generate private key file at path specified in --key-file",
				Destination: &options.GenerateKey,
			},
			&cli.BoolFlag{
				Name:        "overwrite",
				Usage:       "When generating a keyfile, overwrite the nodekey file if it already exists",
				Destination: &options.Overwrite,
			},
			&cli.StringSliceFlag{
				Name:        "staticnode",
				Usage:       "Multiaddr of peer to directly connect with. Option may be repeated",
				Destination: &options.StaticNodes,
			},
			&cli.IntFlag{
				Name:        "keep-alive",
				Value:       20,
				Usage:       "Interval in seconds for pinging peers to keep the connection alive.",
				Destination: &options.KeepAlive,
			},
			&cli.BoolFlag{
				Name:        "use-db",
				Aliases:     []string{"sqlite-store"},
				Usage:       "Use SQLiteDB to persist information",
				Destination: &options.UseDB,
			},
			&cli.BoolFlag{
				Name:        "persist-messages",
				Usage:       "Enable message persistence",
				Destination: &options.Store.PersistMessages,
				Value:       false,
			},
			&cli.BoolFlag{
				Name:        "persist-peers",
				Usage:       "Enable peer persistence",
				Destination: &options.PersistPeers,
				Value:       false,
			},
			&cli.StringFlag{
				Name:        "nat", // This was added so js-waku test don't fail
				Usage:       "TODO: Not implemented yet. Specify method to use for determining public address: any, none ('any' will attempt upnp/pmp)",
				Value:       "any",
				Destination: &options.NAT, // TODO: accept none,any,upnp,extaddr
			},
			&cli.StringFlag{
				Name:        "db-path",
				Aliases:     []string{"dbpath"},
				Value:       "./store.db",
				Usage:       "Path to DB file",
				Destination: &options.DBPath,
			},
			&cli.StringFlag{
				Name:        "advertise-address",
				Usage:       "External address to advertise to other nodes (overrides --address and --ws-address flags)",
				Destination: &options.AdvertiseAddress,
			},
			&cli.BoolFlag{
				Name:        "show-addresses",
				Usage:       "Display listening addresses according to current configuration",
				Destination: &options.ShowAddresses,
			},
			&cli.StringFlag{
				Name:        "log-level",
				Aliases:     []string{"l"},
				Value:       "INFO",
				Usage:       "Define the logging level, supported strings are: DEBUG, INFO, WARN, ERROR, DPANIC, PANIC, FATAL, and their lower-case forms.",
				Destination: &options.LogLevel,
			},
			&cli.BoolFlag{
				Name:        "version",
				Value:       false,
				Usage:       "prints the version",
				Destination: &options.Version,
			},
			&cli.StringFlag{
				Name:        "log-encoding",
				Value:       "console",
				Usage:       "Define the encoding used for the logs: console, json",
				Destination: &options.LogEncoding,
			},
			&cli.BoolFlag{
				Name:        "relay",
				Value:       true,
				Usage:       "Enable relay protocol",
				Destination: &options.Relay.Enable,
			},
			&cli.StringSliceFlag{
				Name:        "topics",
				Usage:       "List of topics to listen",
				Destination: &options.Relay.Topics,
			},
			&cli.BoolFlag{
				Name:        "relay-peer-exchange",
				Aliases:     []string{"peer-exchange"},
				Value:       false,
				Usage:       "Enable GossipSub Peer Exchange",
				Destination: &options.Relay.PeerExchange,
			},
			&cli.IntFlag{
				Name:        "min-relay-peers-to-publish",
				Value:       1,
				Usage:       "Minimum number of peers to publish to Relay",
				Destination: &options.Relay.MinRelayPeersToPublish,
			},
			&cli.BoolFlag{
				Name:        "store",
				Usage:       "Enable relay protocol",
				Destination: &options.Store.Enable,
			},
			&cli.BoolFlag{
				Name:        "resume",
				Usage:       "Fix the gaps in message history",
				Destination: &options.Store.ShouldResume,
			},
			&cli.IntFlag{
				Name:        "store-seconds",
				Value:       (86400 * 30), // 30 days
				Usage:       "maximum number of seconds before a message is removed from the store",
				Destination: &options.Store.RetentionMaxSeconds,
			},
			&cli.IntFlag{
				Name:        "store-capacity",
				Value:       50000,
				Usage:       "maximum number of messages to store",
				Destination: &options.Store.RetentionMaxMessages,
			},
			&cli.StringSliceFlag{
				Name:        "storenode",
				Usage:       "Multiaddr of a peer that supports store protocol. Option may be repeated",
				Destination: &options.Store.Nodes,
			},
			&cli.BoolFlag{
				Name:        "swap",
				Usage:       "Enable swap protocol",
				Value:       false,
				Destination: &options.Swap.Enable,
			},
			&cli.IntFlag{
				Name:        "swap-mode",
				Value:       0,
				Usage:       "Swap mode: 0=soft, 1=mock, 2=hard",
				Destination: &options.Swap.Mode,
			},
			&cli.IntFlag{
				Name:        "swap-payment-threshold",
				Value:       100,
				Usage:       "Threshold for payment",
				Destination: &options.Swap.PaymentThreshold,
			},
			&cli.IntFlag{
				Name:        "swap-disconnect-threshold",
				Value:       -100,
				Usage:       "Threshold for disconnecting",
				Destination: &options.Swap.DisconnectThreshold,
			},
			&cli.BoolFlag{
				Name:        "filter",
				Usage:       "Enable filter protocol",
				Destination: &options.Filter.Enable,
			},
			&cli.BoolFlag{
				Name:        "light-client",
				Usage:       "Don't accept filter subscribers",
				Destination: &options.Filter.DisableFullNode,
			},
			&cli.StringSliceFlag{
				Name:        "filternode",
				Usage:       "Multiaddr of a peer that supports filter protocol. Option may be repeated",
				Destination: &options.Filter.Nodes,
			},
			&cli.IntFlag{
				Name:        "filter-timeout",
				Value:       14400,
				Usage:       "Timeout for filter node in seconds",
				Destination: &options.Filter.Timeout,
			},
			&cli.BoolFlag{
				Name:        "lightpush",
				Usage:       "Enable lightpush protocol",
				Destination: &options.LightPush.Enable,
			},
			&cli.StringSliceFlag{
				Name:        "lightpushnode",
				Usage:       "Multiaddr of a peer that supports lightpush protocol. Option may be repeated",
				Destination: &options.LightPush.Nodes,
			},
			&cli.BoolFlag{
				Name:        "rln-relay",
				Value:       false,
				Usage:       "Enable spam protection through rln-relay",
				Destination: &options.RLNRelay.Enable,
			},
			&cli.IntFlag{
				Name:        "rln-relay-membership-index",
				Value:       0,
				Usage:       "(experimental) the index of node in the rln-relay group: a value between 0-99 inclusive",
				Destination: &options.RLNRelay.MembershipIndex,
			},
			&cli.StringFlag{
				Name:        "rln-relay-pubsub-topic",
				Value:       "/waku/2/default-waku/proto",
				Usage:       "the pubsub topic for which rln-relay gets enabled",
				Destination: &options.RLNRelay.PubsubTopic,
			},
			&cli.StringFlag{
				Name:        "rln-relay-content-topic",
				Value:       "/toy-chat/2/luzhou/proto",
				Usage:       "the content topic for which rln-relay gets enabled",
				Destination: &options.RLNRelay.ContentTopic,
			},
			&cli.BoolFlag{
				Name:        "rln-relay-dynamic",
				Usage:       "Enable waku-rln-relay with on-chain dynamic group management",
				Destination: &options.RLNRelay.Dynamic,
			},
			&cli.StringFlag{
				Name:        "rln-relay-id",
				Usage:       "Rln relay identity secret key as a Hex string",
				Destination: &options.RLNRelay.IDKey,
			},
			&cli.StringFlag{
				Name:        "rln-relay-id-commitment",
				Usage:       "Rln relay identity commitment key as a Hex string",
				Destination: &options.RLNRelay.IDCommitment,
			},
			&cli.StringFlag{
				Name:        "eth-account-address",
				Usage:       "Ethereum testnet account address",
				Destination: &options.RLNRelay.ETHAccount,
			},
			&cli.StringFlag{
				Name:        "eth-client-address",
				Usage:       "Ethereum testnet client address",
				Value:       "ws://localhost:8540",
				Destination: &options.RLNRelay.ETHClientAddress,
			},
			&cli.StringFlag{
				Name:        "eth-mem-contract-address",
				Usage:       "Address of membership contract on an Ethereum testnet",
				Destination: &options.RLNRelay.MembershipContractAddress,
			},
			&cli.BoolFlag{
				Name:        "discv5-discovery",
				Usage:       "Enable discovering nodes via Node Discovery v5",
				Destination: &options.DiscV5.Enable,
			},
			&cli.StringSliceFlag{
				Name:        "discv5-bootstrap-node",
				Usage:       "Text-encoded ENR for bootstrap node. Used when connecting to the network. Option may be repeated",
				Destination: &options.DiscV5.Nodes,
			},
			&cli.IntFlag{
				Name:        "discv5-udp-port",
				Value:       9000,
				Usage:       "Listening UDP port for Node Discovery v5.",
				Destination: &options.DiscV5.Port,
			},
			&cli.BoolFlag{
				Name:        "discv5-enr-auto-update",
				Usage:       "Discovery can automatically update its ENR with the IP address as seen by other nodes it communicates with.",
				Destination: &options.DiscV5.AutoUpdate,
			},
			&cli.BoolFlag{
				Name:        "rendezvous",
				Usage:       "Enable rendezvous protocol for peer discovery",
				Destination: &options.Rendezvous.Enable,
			},
			&cli.StringSliceFlag{
				Name:        "rendezvous-node",
				Usage:       "Multiaddr of a waku2 rendezvous node. Option may be repeated",
				Destination: &options.Rendezvous.Nodes,
			},
			&cli.BoolFlag{
				Name:        "rendezvous-server",
				Usage:       "Node will act as rendezvous server",
				Destination: &options.RendezvousServer.Enable,
			},
			&cli.StringFlag{
				Name:        "rendezvous-db-path",
				Value:       "/tmp/rendezvous",
				Usage:       "Path where peer records database will be stored",
				Destination: &options.RendezvousServer.DBPath,
			},
			&cli.BoolFlag{
				Name:        "dns-discovery",
				Usage:       "Enable DNS discovery",
				Destination: &options.DNSDiscovery.Enable,
			},
			&cli.StringFlag{
				Name:        "dns-discovery-url",
				Usage:       "URL for DNS node list in format 'enrtree://<key>@<fqdn>'",
				Destination: &options.DNSDiscovery.URL,
			},
			&cli.StringFlag{
				Name:        "dns-discovery-name-server",
				Aliases:     []string{"dns-discovery-nameserver"},
				Usage:       "DNS nameserver IP to query (empty to use system's default)",
				Destination: &options.DNSDiscovery.Nameserver,
			},
			&cli.BoolFlag{
				Name:        "metrics-server",
				Aliases:     []string{"metrics"},
				Usage:       "Enable the metrics server",
				Destination: &options.Metrics.Enable,
			},
			&cli.StringFlag{
				Name:        "metrics-server-address",
				Aliases:     []string{"metrics-address"},
				Value:       "127.0.0.1",
				Usage:       "Listening address of the metrics server",
				Destination: &options.Metrics.Address,
			},
			&cli.IntFlag{
				Name:        "metrics-server-port",
				Aliases:     []string{"metrics-port"},
				Value:       8008,
				Usage:       "Listening HTTP port of the metrics server",
				Destination: &options.Metrics.Port,
			},
			&cli.BoolFlag{
				Name:        "rpc",
				Usage:       "Enable the rpc server",
				Destination: &options.RPCServer.Enable,
			},
			&cli.IntFlag{
				Name:        "rpc-port",
				Value:       8545,
				Usage:       "Listening port of the rpc server",
				Destination: &options.RPCServer.Port,
			},
			&cli.StringFlag{
				Name:        "rpc-address",
				Value:       "127.0.0.1",
				Usage:       "Listening address of the rpc server",
				Destination: &options.RPCServer.Address,
			},
			&cli.BoolFlag{
				Name:        "rpc-admin",
				Value:       false,
				Usage:       "Enable access to JSON-RPC Admin API",
				Destination: &options.RPCServer.Admin,
			},
			&cli.BoolFlag{
				Name:        "rpc-private",
				Value:       false,
				Usage:       "Enable access to JSON-RPC Private API",
				Destination: &options.RPCServer.Private,
			},
		},
		Action: func(c *cli.Context) error {
			// for go-libp2p loggers
			lvl, err := logging.LevelFromString(options.LogLevel)
			if err != nil {
				return err
			}
			logging.SetAllLoggers(lvl)

			// go-waku logger
			err = utils.SetLogLevel(options.LogLevel)
			if err != nil {
				return err
			}

			// Set encoding for logs (console, json, ...)
			// Note that libp2p reads the encoding from GOLOG_LOG_FMT env var.
			utils.InitLogger(options.LogEncoding)

			waku.Execute(options)
			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		panic(err)
	}
}
