package server

import (
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/filter"
	"github.com/waku-org/go-waku/waku/v2/protocol/legacy_filter"
	"github.com/waku-org/go-waku/waku/v2/protocol/lightpush"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/protocol/store"
)

func IsWakuProtocol(protocol protocol.ID) bool {
	return protocol == legacy_filter.FilterID_v20beta1 ||
		protocol == filter.FilterPushID_v20beta1 ||
		protocol == filter.FilterSubscribeID_v20beta1 ||
		protocol == relay.WakuRelayID_v200 ||
		protocol == lightpush.LightPushID_v20beta1 ||
		protocol == store.StoreID_v20beta4
}
