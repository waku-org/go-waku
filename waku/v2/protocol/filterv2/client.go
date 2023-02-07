package filterv2

import libp2pProtocol "github.com/libp2p/go-libp2p/core/protocol"

// FilterPushID_v20beta1 is the current Waku Filter protocol identifier used to allow
// filter service nodes to push messages matching registered subscriptions to this client.
const FilterPushID_v20beta1 = libp2pProtocol.ID("/vac/waku/filter-push/2.0.0-beta1")
