package main

import (
	"os"

	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"
	"github.com/waku-org/go-waku/waku"
	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

var options waku.Options

func main() {
	// Defaults
	options.LogLevel = "INFO"
	options.LogEncoding = "console"

	cliFlags := []cli.Flag{
		TcpPort,
		Address,
		WebsocketSupport,
		WebsocketPort,
		WebsocketSecurePort,
		WebsocketAddress,
		WebsocketSecureSupport,
		WebsocketSecureKeyPath,
		WebsocketSecureCertPath,
		Dns4DomainName,
		NodeKey,
		KeyFile,
		KeyPassword,
		GenerateKey,
		Overwrite,
		StaticNode,
		KeepAlive,
		PersistPeers,
		NAT,
		AdvertiseAddress,
		ShowAddresses,
		LogLevel,
		LogEncoding,
		LogOutput,
		AgentString,
		Relay,
		Topics,
		RelayPeerExchange,
		MinRelayPeersToPublish,
		StoreNodeFlag,
		StoreFlag,
		StoreMessageRetentionTime,
		StoreMessageRetentionCapacity,
		StoreResumePeer,
		SwapFlag,
		SwapMode,
		SwapPaymentThreshold,
		SwapDisconnectThreshold,
		FilterFlag,
		LightClient,
		FilterNode,
		FilterTimeout,
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
		MetricsServer,
		MetricsServerAddress,
		MetricsServerPort,
		RPCFlag,
		RPCPort,
		RPCAddress,
		RPCRelayCacheCapacity,
		RPCAdmin,
		RPCPrivate,
		RESTFlag,
		RESTAddress,
		RESTPort,
		RESTRelayCacheCapacity,
		RESTAdmin,
		RESTPrivate,
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
		Flags:   cliFlags,
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
