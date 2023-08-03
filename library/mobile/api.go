// Package gowaku implements gomobile bindings for go-waku. Contains a set of functions that
// are exported when go-waku is exported as libraries for mobile devices
package gowaku

import (
	"github.com/waku-org/go-waku/library"
	"github.com/waku-org/go-waku/waku/v2/protocol"
)

// NewNode initializes a waku node. Receives a JSON string containing the configuration, and use default values for those config items not specified
func NewNode(configJSON string) string {
	err := library.NewNode(configJSON)
	return makeJSONResponse(err)
}

// Start starts the waku node
func Start() string {
	err := library.Start()
	return makeJSONResponse(err)
}

// Stop stops a waku node
func Stop() string {
	err := library.Stop()
	return makeJSONResponse(err)
}

// IsStarted is used to determine is a node is started or not
func IsStarted() string {
	return prepareJSONResponse(library.IsStarted(), nil)
}

// PeerID is used to obtain the peer ID of the waku node
func PeerID() string {
	peerID, err := library.PeerID()
	return prepareJSONResponse(peerID, err)
}

// ListenAddresses returns the multiaddresses the wakunode is listening to
func ListenAddresses() string {
	addresses, err := library.ListenAddresses()
	return prepareJSONResponse(addresses, err)
}

// AddPeer adds a node multiaddress and protocol to the wakunode peerstore
func AddPeer(address string, protocolID string) string {
	peerID, err := library.AddPeer(address, protocolID)
	return prepareJSONResponse(peerID, err)
}

// Connect is used to connect to a peer at multiaddress. if ms > 0, cancel the function execution if it takes longer than N milliseconds
func Connect(address string, ms int) string {
	err := library.Connect(address, ms)
	return makeJSONResponse(err)
}

// ConnectPeerID is usedd to connect to a known peer by peerID. if ms > 0, cancel the function execution if it takes longer than N milliseconds
func ConnectPeerID(peerID string, ms int) string {
	err := library.ConnectPeerID(peerID, ms)
	return makeJSONResponse(err)
}

// Disconnect closes a connection to a known peer by peerID
func Disconnect(peerID string) string {
	err := library.Disconnect(peerID)
	return makeJSONResponse(err)
}

// PeerCnt returns the number of connected peers
func PeerCnt() string {
	peerCnt, err := library.PeerCnt()
	return prepareJSONResponse(peerCnt, err)
}

// ContentTopic creates a content topic string according to RFC 23
func ContentTopic(applicationName string, applicationVersion int, contentTopicName string, encoding string) string {
	return protocol.NewContentTopic(applicationName, uint(applicationVersion), contentTopicName, encoding).String()
}

// PubsubTopic creates a pubsub topic string according to RFC 23
func PubsubTopic(name string, encoding string) string {
	return protocol.NewNamedShardingPubsubTopic(name + "/" + encoding).String()
}

// DefaultPubsubTopic returns the default pubsub topic used in waku2: /waku/2/default-waku/proto
func DefaultPubsubTopic() string {
	return protocol.DefaultPubsubTopic().String()
}

// Peers retrieves the list of peers known by the waku node
func Peers() string {
	peers, err := library.Peers()
	return prepareJSONResponse(peers, err)
}

// DecodeSymmetric decodes a waku message using a 32 bytes symmetric key. The key must be a hex encoded string with "0x" prefix
func DecodeSymmetric(messageJSON string, symmetricKey string) string {
	response, err := library.DecodeSymmetric(messageJSON, symmetricKey)
	return prepareJSONResponse(response, err)
}

// DecodeAsymmetric decodes a waku message using a secp256k1 private key. The key must be a hex encoded string with "0x" prefix
func DecodeAsymmetric(messageJSON string, privateKey string) string {
	response, err := library.DecodeAsymmetric(messageJSON, privateKey)
	return prepareJSONResponse(response, err)
}
