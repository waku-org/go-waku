package main

import (
	"os"

	cli "github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
	"github.com/waku-org/go-waku/cmd/waku/keygen"
	"github.com/waku-org/go-waku/cmd/waku/rlngenerate"
	"github.com/waku-org/go-waku/waku/v2/node"
)

var options NodeOptions

func main() {
	// Defaults
	options.LogLevel = "INFO"
	options.LogEncoding = "console"

	cliFlags := []cli.Flag{
		&cli.StringFlag{Name: "config-file", Usage: "loads configuration from a TOML file (cmd-line parameters take precedence)"},
		TcpPort,
		Address,
		MaxPeerConnections,
		WebsocketSupport,
		WebsocketPort,
		WebsocketSecurePort,
		WebsocketAddress,
		WebsocketSecureSupport,
		WebsocketSecureKeyPath,
		WebsocketSecureCertPath,
		DNS4DomainName,
		NodeKey,
		KeyFile,
		KeyPassword,
		StaticNode,
		KeepAlive,
		PersistPeers,
		NAT,
		IPAddress,
		ExtMultiaddresses,
		ShowAddresses,
		CircuitRelay,
		ResourceScalingMemoryPercent,
		ResourceScalingFDPercent,
		LogLevel,
		LogEncoding,
		LogOutput,
		AgentString,
		Relay,
		Topics,
		ProtectedTopics,
		RelayPeerExchange,
		MinRelayPeersToPublish,
		StoreNodeFlag,
		StoreFlag,
		StoreMessageDBURL,
		StoreMessageRetentionTime,
		StoreMessageRetentionCapacity,
		StoreMessageDBVacuum,
		StoreMessageDBMigration,
		StoreResumePeer,
		FilterFlag,
		FilterNode,
		FilterTimeout,
		FilterLegacyFlag,
		FilterLegacyNode,
		FilterLegacyLightClient,
		LightPush,
		LightPushNode,
		Discv5Discovery,
		Discv5BootstrapNode,
		Discv5UDPPort,
		Discv5ENRAutoUpdate,
		PeerExchange,
		PeerExchangeNode,
		DNSDiscovery,
		DNSDiscoveryUrl,
		DNSDiscoveryNameServer,
		Rendezvous,
		RendezvousNode,
		RendezvousServer,
		MetricsServer,
		MetricsServerAddress,
		MetricsServerPort,
		RPCFlag,
		RPCPort,
		RPCAddress,
		RPCRelayCacheCapacity,
		RPCAdmin,
		RESTFlag,
		RESTAddress,
		RESTPort,
		RESTRelayCacheCapacity,
		RESTAdmin,
		PProf,
	}

	rlnFlags := rlnFlags()
	cliFlags = append(cliFlags, rlnFlags...)

	cli.VersionFlag = &cli.BoolFlag{
		Name:  "version",
		Usage: "prints the version",
	}

	app := &cli.App{
		Name:    "gowaku",
		Version: node.GetVersionInfo().String(),
		Before:  altsrc.InitInputSourceWithContext(cliFlags, altsrc.NewTomlSourceFromFlagFunc("config-file")),
		Flags:   cliFlags,
		Action: func(c *cli.Context) error {
			Execute(options)
			return nil
		},
		Commands: []*cli.Command{
			&keygen.Command,
			&rlngenerate.Command,
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		panic(err)
	}
}
