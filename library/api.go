// Set of functions that are exported when go-waku is built as a static/dynamic library
package main

/*
#include <stdlib.h>
#include <stddef.h>
*/
import "C"
import (
	"unsafe"

	mobile "github.com/waku-org/go-waku/mobile"
	"github.com/waku-org/go-waku/waku/v2/protocol"
)

func main() {}

// Initialize a waku node. Receives a JSON string containing the configuration
// for the node. It can be NULL. Example configuration:
// ```
// {"host": "0.0.0.0", "port": 60000, "advertiseAddr": "1.2.3.4", "nodeKey": "0x123...567", "keepAliveInterval": 20, "relay": true}
// ```
// All keys are optional. If not specified a default value will be set:
// - host: IP address. Default 0.0.0.0
// - port: TCP port to listen. Default 60000. Use 0 for random
// - advertiseAddr: External IP
// - nodeKey: secp256k1 private key. Default random
// - keepAliveInterval: interval in seconds to ping all peers
// - relay: Enable WakuRelay. Default `true`
// - relayTopics: Array of pubsub topics that WakuRelay will automatically subscribe to when the node starts
// - minPeersToPublish: The minimum number of peers required on a topic to allow broadcasting a message. Default `0`
// - filter: Enable Filter. Default `false`
// - discV5: Enable DiscoveryV5. Default `false`
// - discV5BootstrapNodes: Array of bootstrap nodes ENR
// - discV5UDPPort: UDP port for DiscoveryV5
// - logLevel: Set the log level. Default `INFO`. Allowed values "DEBUG", "INFO", "WARN", "ERROR", "DPANIC", "PANIC", "FATAL"
//
//export waku_new
func waku_new(configJSON *C.char) *C.char {
	response := mobile.NewNode(C.GoString(configJSON))
	return C.CString(response)
}

// Starts the waku node
//
//export waku_start
func waku_start() *C.char {
	response := mobile.Start()
	return C.CString(response)
}

// Stops a waku node
//
//export waku_stop
func waku_stop() *C.char {
	response := mobile.Stop()
	return C.CString(response)
}

// Determine is a node is started or not
//
//export waku_is_started
func waku_is_started() *C.char {
	response := mobile.IsStarted()
	return C.CString(response)
}

// Obtain the peer ID of the waku node
//
//export waku_peerid
func waku_peerid() *C.char {
	response := mobile.PeerID()
	return C.CString(response)
}

// Obtain the multiaddresses the wakunode is listening to
//
//export waku_listen_addresses
func waku_listen_addresses() *C.char {
	response := mobile.ListenAddresses()
	return C.CString(response)
}

// Add node multiaddress and protocol to the wakunode peerstore
//
//export waku_add_peer
func waku_add_peer(address *C.char, protocolID *C.char) *C.char {
	response := mobile.AddPeer(C.GoString(address), C.GoString(protocolID))
	return C.CString(response)
}

// Connect to peer at multiaddress. if ms > 0, cancel the function execution if it takes longer than N milliseconds
//
//export waku_connect
func waku_connect(address *C.char, ms C.int) *C.char {
	response := mobile.Connect(C.GoString(address), int(ms))
	return C.CString(response)
}

// Connect to known peer by peerID. if ms > 0, cancel the function execution if it takes longer than N milliseconds
//
//export waku_connect_peerid
func waku_connect_peerid(peerID *C.char, ms C.int) *C.char {
	response := mobile.ConnectPeerID(C.GoString(peerID), int(ms))
	return C.CString(response)
}

// Close connection to a known peer by peerID
//
//export waku_disconnect
func waku_disconnect(peerID *C.char) *C.char {
	response := mobile.Disconnect(C.GoString(peerID))
	return C.CString(response)
}

// Get number of connected peers
//
//export waku_peer_cnt
func waku_peer_cnt() *C.char {
	response := mobile.PeerCnt()
	return C.CString(response)
}

// Create a content topic string according to RFC 23
//
//export waku_content_topic
func waku_content_topic(applicationName *C.char, applicationVersion C.uint, contentTopicName *C.char, encoding *C.char) *C.char {
	return C.CString(protocol.NewContentTopic(C.GoString(applicationName), uint(applicationVersion), C.GoString(contentTopicName), C.GoString(encoding)).String())
}

// Create a pubsub topic string according to RFC 23
//
//export waku_pubsub_topic
func waku_pubsub_topic(name *C.char, encoding *C.char) *C.char {
	return C.CString(mobile.PubsubTopic(C.GoString(name), C.GoString(encoding)))
}

// Get the default pubsub topic used in waku2: /waku/2/default-waku/proto
//
//export waku_default_pubsub_topic
func waku_default_pubsub_topic() *C.char {
	return C.CString(mobile.DefaultPubsubTopic())
}

// Register callback to act as signal handler and receive application signals
// (in JSON) which are used to react to asynchronous events in waku. The function
// signature for the callback should be `void myCallback(char* signalJSON)`
//
//export waku_set_event_callback
func waku_set_event_callback(cb unsafe.Pointer) {
	mobile.SetEventCallback(cb)
}

// Retrieve the list of peers known by the waku node
//
//export waku_peers
func waku_peers() *C.char {
	response := mobile.Peers()
	return C.CString(response)
}

// Decode a waku message using a 32 bytes symmetric key. The key must be a hex encoded string with "0x" prefix
//
//export waku_decode_symmetric
func waku_decode_symmetric(messageJSON *C.char, symmetricKey *C.char) *C.char {
	response := mobile.DecodeSymmetric(C.GoString(messageJSON), C.GoString(symmetricKey))
	return C.CString(response)
}

// Decode a waku message using a secp256k1 private key. The key must be a hex encoded string with "0x" prefix
//
//export waku_decode_asymmetric
func waku_decode_asymmetric(messageJSON *C.char, privateKey *C.char) *C.char {
	response := mobile.DecodeAsymmetric(C.GoString(messageJSON), C.GoString(privateKey))
	return C.CString(response)
}
