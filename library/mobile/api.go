// Implements gomobile bindings for go-waku. Contains a set of functions that
// are exported when go-waku is exported as libraries for mobile devices
package gowaku

import (
	"github.com/waku-org/go-waku/library"
	"github.com/waku-org/go-waku/waku/v2/protocol"
)

func NewNode(configJSON string) string {
	err := library.NewNode(configJSON)
	return MakeJSONResponse(err)
}

func Start() string {
	err := library.Start()
	return MakeJSONResponse(err)
}

func Stop() string {
	err := library.Stop()
	return MakeJSONResponse(err)
}

func IsStarted() string {
	return PrepareJSONResponse(library.IsStarted(), nil)
}

func PeerID() string {
	peerID, err := library.PeerID()
	return PrepareJSONResponse(peerID, err)
}

func ListenAddresses() string {
	addresses, err := library.ListenAddresses()
	return PrepareJSONResponse(addresses, err)
}

func AddPeer(address string, protocolID string) string {
	peerID, err := library.AddPeer(address, protocolID)
	return PrepareJSONResponse(peerID, err)
}

func Connect(address string, ms int) string {
	err := library.Connect(address, ms)
	return MakeJSONResponse(err)
}

func ConnectPeerID(peerID string, ms int) string {
	err := library.ConnectPeerID(peerID, ms)
	return MakeJSONResponse(err)
}

func Disconnect(peerID string) string {
	err := library.Disconnect(peerID)
	return MakeJSONResponse(err)
}

func PeerCnt() string {
	peerCnt, err := library.PeerCnt()
	return PrepareJSONResponse(peerCnt, err)
}

func ContentTopic(applicationName string, applicationVersion int, contentTopicName string, encoding string) string {
	return protocol.NewContentTopic(applicationName, uint(applicationVersion), contentTopicName, encoding).String()
}

func PubsubTopic(name string, encoding string) string {
	return protocol.NewNamedShardingPubsubTopic(name + "/" + encoding).String()
}

func DefaultPubsubTopic() string {
	return protocol.DefaultPubsubTopic().String()
}

func Peers() string {
	peers, err := library.Peers()
	return PrepareJSONResponse(peers, err)
}

func DecodeSymmetric(messageJSON string, symmetricKey string) string {
	response, err := library.DecodeSymmetric(messageJSON, symmetricKey)
	return PrepareJSONResponse(response, err)
}

func DecodeAsymmetric(messageJSON string, privateKey string) string {
	response, err := library.DecodeAsymmetric(messageJSON, privateKey)
	return PrepareJSONResponse(response, err)
}
