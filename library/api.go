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
// - gossipsubParams: an object containing custom gossipsub parameters. All attributes are optional, and if not specified, it will use default values.
//   - d: optimal degree for a GossipSub topic mesh. Default `6`
//   - dLow: lower bound on the number of peers we keep in a GossipSub topic mesh. Default `5`
//   - dHigh: upper bound on the number of peers we keep in a GossipSub topic mesh. Default `12`
//   - dScore: affects how peers are selected when pruning a mesh due to over subscription. Default `4`
//   - dOut: sets the quota for the number of outbound connections to maintain in a topic mesh. Default `2`
//   - historyLength: controls the size of the message cache used for gossip. Default `5`
//   - historyGossip: controls how many cached message ids we will advertise in IHAVE gossip messages. Default `3`
//   - dLazy: affects how many peers we will emit gossip to at each heartbeat. Default `6`
//   - gossipFactor: affects how many peers we will emit gossip to at each heartbeat. Default `0.25`
//   - gossipRetransmission: controls how many times we will allow a peer to request the same message id through IWANT gossip before we start ignoring them. Default `3`
//   - heartbeatInitialDelayMs: short delay in milliseconds before the heartbeat timer begins after the router is initialized. Default `100` milliseconds
//   - heartbeatIntervalSeconds: controls the time between heartbeats. Default `1` second
//   - slowHeartbeatWarning: duration threshold for heartbeat processing before emitting a warning. Default `0.1`
//   - fanoutTTLSeconds: controls how long we keep track of the fanout state. Default `60` seconds
//   - prunePeers: controls the number of peers to include in prune Peer eXchange. Default `16`
//   - pruneBackoffSeconds: controls the backoff time for pruned peers. Default `60` seconds
//   - unsubscribeBackoffSeconds: controls the backoff time to use when unsuscribing from a topic. Default `10` seconds
//   - connectors: number of active connection attempts for peers obtained through PX. Default `8`
//   - maxPendingConnections: maximum number of pending connections for peers attempted through px. Default `128`
//   - connectionTimeoutSeconds: timeout in seconds for connection attempts. Default `30` seconds
//   - directConnectTicks: the number of heartbeat ticks for attempting to reconnect direct peers that are not currently connected. Default `300`
//   - directConnectInitialDelaySeconds: initial delay before opening connections to direct peers. Default `1` second
//   - opportunisticGraftTicks: number of heartbeat ticks for attempting to improve the mesh with opportunistic grafting. Default `60`
//   - opportunisticGraftPeers: the number of peers to opportunistically graft. Default `2`
//   - graftFloodThresholdSeconds: If a GRAFT comes before GraftFloodThresholdSeconds has elapsed since the last PRUNE, then there is an extra score penalty applied to the peer through P7. Default `10` seconds
//   - maxIHaveLength: max number of messages to include in an IHAVE message, also controls the max number of IHAVE ids we will accept and request with IWANT from a peer within a heartbeat. Default `5000`
//   - maxIHaveMessages: max number of IHAVE messages to accept from a peer within a heartbeat. Default `10`
//   - iWantFollowupTimeSeconds: Time to wait for a message requested through IWANT following an IHAVE advertisement. Default `3` seconds
//   - seenMessagesTTLSeconds:  configures when a previously seen message ID can be forgotten about. Default `120` seconds
//
// - minPeersToPublish: The minimum number of peers required on a topic to allow broadcasting a message. Default `0`
// - legacyFilter: Enable LegacyFilter. Default `false`
// - discV5: Enable DiscoveryV5. Default `false`
// - discV5BootstrapNodes: Array of bootstrap nodes ENR
// - discV5UDPPort: UDP port for DiscoveryV5
// - logLevel: Set the log level. Default `INFO`. Allowed values "DEBUG", "INFO", "WARN", "ERROR", "DPANIC", "PANIC", "FATAL"
// - store: Enable Store. Default `false`
// - databaseURL: url connection string. Default: "sqlite3://store.db". Also accepts PostgreSQL connection strings
// - storeRetentionMaxMessages: max number of messages to store in the database. Default 10000
// - storeRetentionTimeSeconds: max number of seconds that a message will be persisted in the database. Default 2592000 (30d)
// - websockets: an optional object containing settings to setup the websocket configuration
//   - enabled: indicates if websockets support will be enabled. Default `false`
//   - host: listening address for websocket connections. Default `0.0.0.0`
//   - port: TCP listening port for websocket connection (0 for random, binding to 443 requires root access). Default: `60001“, if secure websockets support is enabled, the default is `6443“
//   - secure: enable secure websockets support. Default `false`
//   - certPath: secure websocket certificate path
//   - keyPath: secure websocket key path
//
// - dns4DomainName: the domain name resolving to the node's public IPv4 address.
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
