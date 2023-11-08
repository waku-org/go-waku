package main

import (
	"fmt"

	"github.com/waku-org/go-waku/waku/cliutils"
	wcli "github.com/waku-org/go-waku/waku/cliutils"
	"github.com/waku-org/go-waku/waku/v2/protocol"

	"github.com/urfave/cli/v2"
)

type FleetValue struct {
	Value   *Fleet
	Default Fleet
}

func (v *FleetValue) Set(value string) error {
	if value == string(fleetProd) || value == string(fleetTest) || value == string(fleetNone) {
		*v.Value = Fleet(value)
		return nil
	}
	return fmt.Errorf("%s is not a valid option. need %+v", value, []Fleet{fleetProd, fleetTest, fleetNone})
}

func (v *FleetValue) String() string {
	if v.Value == nil {
		return string(v.Default)
	}
	return string(*v.Value)
}

func getFlags() []cli.Flag {
	// Defaults
	options.Fleet = fleetProd

	testCT, err := protocol.NewContentTopic("toy-chat", "3", "mingde", "proto")
	if err != nil {
		panic("invalid contentTopic")
	}
	testnetContentTopic := testCT.String()

	return []cli.Flag{
		&cli.GenericFlag{
			Name:  "nodekey",
			Usage: "P2P node private key as hex. (default random)",
			Value: &wcli.PrivateKeyValue{
				Value: &options.NodeKey,
			},
		},
		&cli.StringFlag{
			Name:        "listen-address",
			Aliases:     []string{"host", "address"},
			Value:       "0.0.0.0",
			Usage:       "Listening address",
			Destination: &options.Address,
		},
		&cli.IntFlag{
			Name:        "tcp-port",
			Aliases:     []string{"port", "p"},
			Value:       0,
			Usage:       "Libp2p TCP listening port (0 for random)",
			Destination: &options.Port,
		},
		&cli.IntFlag{
			Name:        "udp-port",
			Value:       60000,
			Usage:       "Listening UDP port for Node Discovery v5.",
			Destination: &options.DiscV5.Port,
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
		&cli.StringFlag{
			Name:        "content-topic",
			Usage:       "content topic to use for the chat",
			Value:       testnetContentTopic,
			Destination: &options.ContentTopic,
		},
		&cli.GenericFlag{
			Name:  "fleet",
			Usage: "Select the fleet to connect to",
			Value: &FleetValue{
				Default: fleetProd,
				Value:   &options.Fleet,
			},
		},
		&cli.GenericFlag{
			Name:  "staticnode",
			Usage: "Multiaddr of peer to directly connect with. Option may be repeated",
			Value: &wcli.MultiaddrSlice{
				Values: &options.StaticNodes,
			},
		},
		&cli.StringFlag{
			Name:        "nickname",
			Usage:       "nickname to use in chat.",
			Destination: &options.Nickname,
			Value:       "Anonymous",
		},
		&cli.BoolFlag{
			Name:        "relay",
			Value:       true,
			Usage:       "Enable relay protocol",
			Destination: &options.Relay.Enable,
		},
		&cli.StringSliceFlag{
			Name:        "topic",
			Usage:       "Pubsub topics to subscribe to. Option can be repeated",
			Destination: &options.Relay.Topics,
		},
		&cli.BoolFlag{
			Name:        "store",
			Usage:       "Enable store protocol",
			Value:       true,
			Destination: &options.Store.Enable,
		},
		&cli.GenericFlag{
			Name:  "storenode",
			Usage: "Multiaddr of a peer that supports store protocol.",
			Value: &wcli.MultiaddrValue{
				Value: &options.Store.Node,
			},
		},
		&cli.BoolFlag{
			Name:        "filter",
			Usage:       "Enable filter protocol",
			Destination: &options.Filter.Enable,
		},
		&cli.GenericFlag{
			Name:  "filternode",
			Usage: "Multiaddr of a peer that supports filter protocol.",
			Value: &wcli.MultiaddrValue{
				Value: &options.Filter.Node,
			},
		},
		&cli.BoolFlag{
			Name:        "lightpush",
			Usage:       "Enable lightpush protocol",
			Destination: &options.LightPush.Enable,
		},
		&cli.GenericFlag{
			Name:  "lightpushnode",
			Usage: "Multiaddr of a peer that supports lightpush protocol.",
			Value: &wcli.MultiaddrValue{
				Value: &options.LightPush.Node,
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
		&cli.BoolFlag{
			Name:        "discv5-enr-auto-update",
			Usage:       "Discovery can automatically update its ENR with the IP address as seen by other nodes it communicates with.",
			Destination: &options.DiscV5.AutoUpdate,
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
			Name:        "rln-relay",
			Value:       false,
			Usage:       "Enable spam protection through rln-relay",
			Destination: &options.RLNRelay.Enable,
		},
		&cli.GenericFlag{
			Name:  "rln-relay-cred-index",
			Usage: "the index of the onchain commitment to use",
			Value: &wcli.OptionalUint{
				Value: &options.RLNRelay.MembershipIndex,
			},
		},
		&cli.BoolFlag{
			Name:        "rln-relay-dynamic",
			Usage:       "Enable waku-rln-relay with on-chain dynamic group management",
			Destination: &options.RLNRelay.Dynamic,
		},
		&cli.PathFlag{
			Name:        "rln-relay-cred-path",
			Usage:       "The path for persisting rln-relay credential",
			Value:       "",
			Destination: &options.RLNRelay.CredentialsPath,
		},
		&cli.StringFlag{
			Name:        "rln-relay-cred-password",
			Value:       "",
			Usage:       "Password for encrypting RLN credentials",
			Destination: &options.RLNRelay.CredentialsPassword,
		},
		&cli.StringFlag{
			Name:        "rln-relay-eth-client-address",
			Usage:       "Ethereum testnet client address",
			Value:       "ws://localhost:8545",
			Destination: &options.RLNRelay.ETHClientAddress,
		},
		&cli.GenericFlag{
			Name:  "rln-relay-eth-contract-address",
			Usage: "Address of membership contract on an Ethereum testnet",
			Value: &wcli.AddressValue{
				Value: &options.RLNRelay.MembershipContractAddress,
			},
		},
	}
}
