package main

import (
	"crypto/ecdsa"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/urfave/cli/v2"
)

// DiscV5Options are settings to enable a modified version of Ethereum’s Node
// Discovery Protocol v5 as a means for ambient node discovery.
type DiscV5Options struct {
	Enable     bool
	Nodes      cli.StringSlice
	Port       int
	AutoUpdate bool
}

// RelayOptions are settings to enable the relay protocol which is a pubsub
// approach to peer-to-peer messaging with a strong focus on privacy,
// censorship-resistance, security and scalability.
type RelayOptions struct {
	Enable bool
	Topics cli.StringSlice
}

type RLNRelayOptions struct {
	Enable                    bool
	CredentialsPath           string
	MembershipIndex           int
	PubsubTopic               string
	ContentTopic              string
	Dynamic                   bool
	ETHPrivateKey             *ecdsa.PrivateKey
	ETHClientAddress          string
	MembershipContractAddress common.Address
}

// FilterOptions are settings used to enable filter protocol. This is a protocol
// that enables subscribing to messages that a peer receives. This is a more
// lightweight version of WakuRelay specifically designed for bandwidth
// restricted devices.
type FilterOptions struct {
	Enable bool
	Node   *multiaddr.Multiaddr
}

func (f FilterOptions) NodePeerID() (peer.ID, error) {
	if f.Node == nil {
		return peer.ID(""), errors.New("node is nil")
	}

	peerID, err := (*f.Node).ValueForProtocol(multiaddr.P_P2P)
	if err != nil {
		return peer.ID(""), err
	}

	return peer.Decode(peerID)
}

// LightpushOptions are settings used to enable the lightpush protocol. This is
// a lightweight protocol used to avoid having to run the relay protocol which
// is more resource intensive. With this protocol a message is pushed to a peer
// that supports both the lightpush protocol and relay protocol. That peer will
// broadcast the message and return a confirmation that the message was
// broadcasted
type LightpushOptions struct {
	Enable bool
	Node   *multiaddr.Multiaddr
}

func (f LightpushOptions) NodePeerID() (peer.ID, error) {
	if f.Node == nil {
		return peer.ID(""), errors.New("node is nil")
	}

	peerID, err := (*f.Node).ValueForProtocol(multiaddr.P_P2P)
	if err != nil {
		return peer.ID(""), err
	}

	return peer.Decode(peerID)
}

// StoreOptions are settings used for enabling the store protocol, used to
// retrieve message history from other nodes
type StoreOptions struct {
	Enable bool
	Node   *multiaddr.Multiaddr
}

func (f StoreOptions) NodePeerID() (peer.ID, error) {
	if f.Node == nil {
		return peer.ID(""), errors.New("node is nil")
	}

	peerID, err := (*f.Node).ValueForProtocol(multiaddr.P_P2P)
	if err != nil {
		return peer.ID(""), err
	}

	return peer.Decode(peerID)
}

// DNSDiscoveryOptions are settings used for enabling DNS-based discovery
// protocol that stores merkle trees in DNS records which contain connection
// information for nodes. It's very useful for bootstrapping a p2p network.
type DNSDiscoveryOptions struct {
	Enable     bool
	URL        string
	Nameserver string
}

type Fleet string

const fleetNone Fleet = "none"
const fleetProd Fleet = "prod"
const fleetTest Fleet = "test"

// Options contains all the available features and settings that can be
// configured via flags when executing chat2
type Options struct {
	Port         int
	Fleet        Fleet
	Address      string
	NodeKey      *ecdsa.PrivateKey
	ContentTopic string
	UsePayloadV1 bool
	Nickname     string
	StaticNodes  []multiaddr.Multiaddr

	Relay        RelayOptions
	Store        StoreOptions
	Filter       FilterOptions
	LightPush    LightpushOptions
	RLNRelay     RLNRelayOptions
	DiscV5       DiscV5Options
	DNSDiscovery DNSDiscoveryOptions
}
