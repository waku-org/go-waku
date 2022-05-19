// logging implements custom logging field types for commonly
// logged values like host ID or wallet address.
//
// implementation purposely does as little as possible at field creation time,
// and postpones any transformation to output time by relying on the generic
// zap types like zap.Stringer, zap.Array, zap.Object
//
package logging

import (
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/status-im/go-waku/waku/v2/protocol/pb"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// List of multiaddrs

type multiaddrs []multiaddr.Multiaddr

func MultiAddrs(key string, addrs ...multiaddr.Multiaddr) zapcore.Field {
	return zap.Array(key, multiaddrs(addrs))
}

func (addrs multiaddrs) MarshalLogArray(encoder zapcore.ArrayEncoder) error {
	for _, addr := range addrs {
		encoder.AppendString(addr.String())
	}
	return nil
}

// Host ID

type hostID peer.ID

func HostID(key string, id peer.ID) zapcore.Field {
	return zap.Stringer(key, hostID(id))
}

func (id hostID) String() string { return peer.Encode(peer.ID(id)) }

// Time - looks like Waku is using Microsecond Unix Time

type timestamp int64

func Time(key string, time int64) zapcore.Field {
	return zap.Stringer(key, timestamp(time))
}

func (t timestamp) String() string {
	return time.UnixMicro(int64(t)).Format(time.RFC3339)
}

// History Query Filters

type historyFilters []*pb.ContentFilter

func Filters(filters []*pb.ContentFilter) zapcore.Field {
	return zap.Array("filters", historyFilters(filters))
}

func (filters historyFilters) MarshalLogArray(encoder zapcore.ArrayEncoder) error {
	for _, filter := range filters {
		encoder.AppendString(filter.ContentTopic)
	}
	return nil
}

// History Paging Info

type pagingInfo pb.PagingInfo
type index pb.Index

func PagingInfo(pi *pb.PagingInfo) zapcore.Field {
	return zap.Object("paging_info", (*pagingInfo)(pi))
}

func (pi *pagingInfo) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	encoder.AddUint64("page_size", pi.PageSize)
	encoder.AddString("direction", pi.Direction.String())
	if pi.Cursor != nil {
		return encoder.AddObject("cursor", (*index)(pi.Cursor))
	}
	return nil
}

func (i *index) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	encoder.AddBinary("digest", i.Digest)
	encoder.AddTime("sent", time.UnixMicro(i.SenderTime))
	encoder.AddTime("received", time.UnixMicro(i.ReceiverTime))
	return nil
}

// Wallet Address

func WalletAddress(address string) zapcore.Field {
	return zap.String("wallet_address", address)
}

// Hex encoded bytes

type hexBytes []byte

func Bytes(key string, bytes []byte) zap.Field {
	return zap.Stringer(key, hexBytes(bytes))
}

func (bytes hexBytes) String() string {
	return hexutil.Encode(bytes)
}
