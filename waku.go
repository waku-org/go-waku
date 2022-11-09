package main

import (
	"os"
	"time"

	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"
	"github.com/waku-org/go-waku/waku"
	"github.com/waku-org/go-waku/waku/cliutils"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

var options waku.Options

func main() {
	// Defaults
	options.LogLevel = "INFO"
	options.LogEncoding = "console"

	cliFlags := []cli.Flag{
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
			Destination: &options.Websocket.WSPort,
		},
		&cli.IntFlag{
			Name:        "websocket-secure-port",
			Aliases:     []string{"wss-port"},
			Value:       6443,
			Usage:       "Libp2p TCP listening port for secure websocket connection (0 for random, binding to 443 requires root access)",
			Destination: &options.Websocket.WSSPort,
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
		&cli.PathFlag{
			Name:        "websocket-secure-key-path",
			Aliases:     []string{"wss-key"},
			Value:       "/path/to/key.txt",
			Usage:       "Secure websocket key path",
			Destination: &options.Websocket.KeyPath,
		},
		&cli.PathFlag{
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
		&cli.GenericFlag{
			Name:  "nodekey",
			Usage: "P2P node private key as hex. Can also be set with GOWAKU-NODEKEY env variable (default random)",
			Value: &cliutils.PrivateKeyValue{
				Value: &options.NodeKey,
			},
			EnvVars: []string{"GOWAKU-NODEKEY"},
		},
		&cli.PathFlag{
			Name:        "key-file",
			Value:       "./nodekey",
			Usage:       "Path to a file containing the private key for the P2P node",
			Destination: &options.KeyFile,
		},
		&cli.StringFlag{
			Name:        "key-password",
			Value:       "secret",
			Usage:       "Password used for the private key file",
			Destination: &options.KeyPasswd,
		},
		&cli.BoolFlag{
			Name:        "generate-key",
			Usage:       "Generate private key file at path specified in --key-file with the password defined by --key-password",
			Destination: &options.GenerateKey,
		},
		&cli.BoolFlag{
			Name:        "overwrite",
			Usage:       "When generating a keyfile, overwrite the nodekey file if it already exists",
			Destination: &options.Overwrite,
		},
		&cli.GenericFlag{
			Name:  "staticnode",
			Usage: "Multiaddr of peer to directly connect with. Option may be repeated",
			Value: &cliutils.MultiaddrSlice{
				Values: &options.StaticNodes,
			},
		},
		&cli.DurationFlag{
			Name:        "keep-alive",
			Value:       20 * time.Second,
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
		&cli.PathFlag{
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
		&cli.GenericFlag{
			Name:    "log-level",
			Aliases: []string{"l"},
			Value: &cliutils.ChoiceValue{
				Choices: []string{"DEBUG", "INFO", "WARN", "ERROR", "DPANIC", "PANIC", "FATAL"},
				Value:   &options.LogLevel,
			},
			Usage: "Define the logging level,",
		},
		&cli.GenericFlag{
			Name:  "log-encoding",
			Usage: "Define the encoding used for the logs",
			Value: &cliutils.ChoiceValue{
				Choices: []string{"console", "nocolor", "json"},
				Value:   &options.LogEncoding,
			},
		},
		&cli.StringFlag{
			Name:        "log-output",
			Value:       "stdout",
			Usage:       "specifies where logging output should be written  (stdout, file, file:./filename.log)",
			Destination: &options.LogOutput,
		},
		&cli.BoolFlag{
			Name:        "version",
			Value:       false,
			Usage:       "prints the version",
			Destination: &options.Version,
		},
		&cli.StringFlag{
			Name:        "agent-string",
			Value:       "go-waku",
			Usage:       "client id to advertise",
			Destination: &options.UserAgent,
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
			Usage:       "Enable store protocol to persist messages",
			Destination: &options.Store.Enable,
		},
		&cli.BoolFlag{
			Name:        "resume",
			Usage:       "Fix the gaps in message history",
			Destination: &options.Store.ShouldResume,
		},
		&cli.DurationFlag{
			Name:        "store-duration",
			Value:       time.Hour * 24 * 30,
			Usage:       "maximum number of seconds before a message is removed from the store",
			Destination: &options.Store.RetentionTime,
		},
		&cli.IntFlag{
			Name:        "store-capacity",
			Value:       50000,
			Usage:       "maximum number of messages to store",
			Destination: &options.Store.RetentionMaxMessages,
		},
		&cli.GenericFlag{
			Name:  "storenode",
			Usage: "Multiaddr of a peer that supports store protocol. Option may be repeated",
			Value: &cliutils.MultiaddrSlice{
				Values: &options.Store.Nodes,
			},
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
		&cli.GenericFlag{
			Name:  "filternode",
			Usage: "Multiaddr of a peer that supports filter protocol. Option may be repeated",
			Value: &cliutils.MultiaddrSlice{
				Values: &options.Filter.Nodes,
			},
		},
		&cli.DurationFlag{
			Name:        "filter-timeout",
			Value:       14400 * time.Second,
			Usage:       "Timeout for filter node in seconds",
			Destination: &options.Filter.Timeout,
		},
		&cli.BoolFlag{
			Name:        "lightpush",
			Usage:       "Enable lightpush protocol",
			Destination: &options.LightPush.Enable,
		},
		&cli.GenericFlag{
			Name:  "lightpushnode",
			Usage: "Multiaddr of a peer that supports lightpush protocol. Option may be repeated",
			Value: &cliutils.MultiaddrSlice{
				Values: &options.LightPush.Nodes,
			},
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
			Name:        "peer-exchange",
			Usage:       "Enable waku peer exchange protocol (responder side)",
			Destination: &options.PeerExchange.Enable,
		},
		&cli.GenericFlag{
			Name:  "peer-exchange-node",
			Usage: "Peer multiaddr to send peer exchange requests to. (enables peer exchange protocol requester side)",
			Value: &cliutils.MultiaddrValue{
				Value: &options.PeerExchange.Node,
			},
		},
		&cli.BoolFlag{
			Name:        "dns-discovery",
			Usage:       "Enable DNS discovery",
			Destination: &options.DNSDiscovery.Enable,
		},
		&cli.StringSliceFlag{
			Name:        "dns-discovery-url",
			Usage:       "URL for DNS node list in format 'enrtree://<key>@<fqdn>'",
			Destination: &options.DNSDiscovery.URLs,
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
		&cli.IntFlag{
			Name:        "rpc-relay-cache-capacity",
			Value:       30,
			Usage:       "Capacity of the Relay REST API message cache",
			Destination: &options.RPCServer.RelayCacheCapacity,
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

		&cli.BoolFlag{
			Name:        "rest",
			Usage:       "Enable Waku REST HTTP server",
			Destination: &options.RESTServer.Enable,
		},
		&cli.StringFlag{
			Name:        "rest-address",
			Value:       "127.0.0.1",
			Usage:       "Listening address of the REST HTTP server",
			Destination: &options.RESTServer.Address,
		},
		&cli.IntFlag{
			Name:        "rest-port",
			Value:       8645,
			Usage:       "Listening port of the REST HTTP server",
			Destination: &options.RESTServer.Port,
		},
		&cli.IntFlag{
			Name:        "rest-relay-cache-capacity",
			Value:       30,
			Usage:       "Capacity of the Relay REST API message cache",
			Destination: &options.RESTServer.RelayCacheCapacity,
		},
		&cli.BoolFlag{
			Name:        "rest-admin",
			Value:       false,
			Usage:       "Enable access to REST HTTP Admin API",
			Destination: &options.RESTServer.Admin,
		},
		&cli.BoolFlag{
			Name:        "rest-private",
			Value:       false,
			Usage:       "Enable access to REST HTTP Private API",
			Destination: &options.RESTServer.Private,
		},
	}

	rlnFlags := rlnFlags()
	cliFlags = append(cliFlags, rlnFlags...)

	app := &cli.App{
		Flags: cliFlags,
		Action: func(c *cli.Context) error {
			utils.InitLogger(options.LogEncoding, options.LogOutput)

			lvl, err := logging.LevelFromString(options.LogLevel)
			if err != nil {
				return err
			}
			logging.SetAllLoggers(lvl)

			waku.Execute(options)
			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		panic(err)
	}
}
