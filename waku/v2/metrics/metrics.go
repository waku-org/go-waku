package metrics

import (
	"context"
	"fmt"
	"time"

	"github.com/waku-org/go-waku/waku/v2/utils"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"go.uber.org/zap"
)

var (
	WakuVersion = stats.Int64("waku_version", "", stats.UnitDimensionless)
	Messages    = stats.Int64("node_messages", "Number of messages received", stats.UnitDimensionless)
	Peers       = stats.Int64("peers", "Number of connected peers", stats.UnitDimensionless)
	Dials       = stats.Int64("dials", "Number of peer dials", stats.UnitDimensionless)

	LegacyFilterMessages      = stats.Int64("legacy_filter_messages", "Number of legacy filter messages", stats.UnitDimensionless)
	LegacyFilterSubscribers   = stats.Int64("legacy_filter_subscribers", "Number of legacy filter subscribers", stats.UnitDimensionless)
	LegacyFilterSubscriptions = stats.Int64("legacy_filter_subscriptions", "Number of legacy filter subscriptions", stats.UnitDimensionless)
	LegacyFilterErrors        = stats.Int64("legacy_filter_errors", "Number of errors in legacy filter protocol", stats.UnitDimensionless)

	FilterMessages                     = stats.Int64("filter_messages", "Number of filter messages", stats.UnitDimensionless)
	FilterRequests                     = stats.Int64("filter_requests", "Number of filter requests", stats.UnitDimensionless)
	FilterSubscriptions                = stats.Int64("filter_subscriptions", "Number of filter subscriptions", stats.UnitDimensionless)
	FilterErrors                       = stats.Int64("filter_errors", "Number of errors in filter protocol", stats.UnitDimensionless)
	FilterRequestDurationSeconds       = stats.Int64("filter_request_duration_seconds", "Duration of Filter Subscribe Requests", stats.UnitSeconds)
	FilterHandleMessageDurationSeconds = stats.Int64("filter_handle_msessageduration_seconds", "Duration to Push Message to Filter Subscribers", stats.UnitSeconds)

	StoredMessages = stats.Int64("store_messages", "Number of historical messages", stats.UnitDimensionless)
	StoreErrors    = stats.Int64("errors", "Number of errors in store protocol", stats.UnitDimensionless)
	StoreQueries   = stats.Int64("store_queries", "Number of store queries", stats.UnitDimensionless)

	LightpushMessages = stats.Int64("lightpush_messages", "Number of messages sent via lightpush protocol", stats.UnitDimensionless)
	LightpushErrors   = stats.Int64("errors", "Number of errors in lightpush protocol", stats.UnitDimensionless)

	PeerExchangeError = stats.Int64("errors", "Number of errors in peer exchange protocol", stats.UnitDimensionless)

	DnsDiscoveryNodes  = stats.Int64("dnsdisc_nodes", "Number of discovered nodes", stats.UnitDimensionless)
	DnsDiscoveryErrors = stats.Int64("errors", "Number of errors in dns discovery", stats.UnitDimensionless)
)

var (
	KeyType, _    = tag.NewKey("type")
	ErrorType, _  = tag.NewKey("error_type")
	GitVersion, _ = tag.NewKey("git_version")
)

var (
	PeersView = &view.View{
		Name:        "gowaku_connected_peers",
		Measure:     Peers,
		Description: "Number of connected peers",
		Aggregation: view.Sum(),
	}
	DialsView = &view.View{
		Name:        "gowaku_peers_dials",
		Measure:     Dials,
		Description: "Number of peer dials",
		Aggregation: view.Count(),
	}
	MessageView = &view.View{
		Name:        "gowaku_node_messages",
		Measure:     Messages,
		Description: "The number of the messages received",
		Aggregation: view.Count(),
	}

	StoreQueriesView = &view.View{
		Name:        "gowaku_store_queries",
		Measure:     StoreQueries,
		Description: "The number of the store queries received",
		Aggregation: view.Count(),
	}
	StoreMessagesView = &view.View{
		Name:        "gowaku_store_messages",
		Measure:     StoredMessages,
		Description: "The distribution of the store protocol messages",
		Aggregation: view.LastValue(),
		TagKeys:     []tag.Key{KeyType},
	}
	StoreErrorTypesView = &view.View{
		Name:        "gowaku_store_errors",
		Measure:     StoreErrors,
		Description: "The distribution of the store protocol errors",
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{ErrorType},
	}

	LegacyFilterSubscriptionsView = &view.View{
		Name:        "gowaku_legacy_filter_subscriptions",
		Measure:     LegacyFilterSubscriptions,
		Description: "The number of legacy filter subscriptions",
		Aggregation: view.Count(),
	}
	LegacyFilterSubscribersView = &view.View{
		Name:        "gowaku_legacy_filter_subscribers",
		Measure:     LegacyFilterSubscribers,
		Description: "The number of legacy filter subscribers",
		Aggregation: view.LastValue(),
	}
	LegacyFilterMessagesView = &view.View{
		Name:        "gowaku_legacy_filter_messages",
		Measure:     LegacyFilterMessages,
		Description: "The distribution of the legacy filter protocol messages received",
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{KeyType},
	}
	LegacyFilterErrorTypesView = &view.View{
		Name:        "gowaku_legacy_filter_errors",
		Measure:     LegacyFilterErrors,
		Description: "The distribution of the legacy filter protocol errors",
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{ErrorType},
	}

	FilterSubscriptionsView = &view.View{
		Name:        "gowaku_filter_subscriptions",
		Measure:     FilterSubscriptions,
		Description: "The number of filter subscriptions",
		Aggregation: view.Count(),
	}
	FilterRequestsView = &view.View{
		Name:        "gowaku_filter_requests",
		Measure:     FilterRequests,
		Description: "The number of filter requests",
		Aggregation: view.Count(),
	}
	FilterMessagesView = &view.View{
		Name:        "gowaku_filter_messages",
		Measure:     FilterMessages,
		Description: "The distribution of the filter protocol messages received",
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{KeyType},
	}
	FilterErrorTypesView = &view.View{
		Name:        "gowaku_filter_errors",
		Measure:     FilterErrors,
		Description: "The distribution of the filter protocol errors",
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{ErrorType},
	}
	FilterRequestDurationView = &view.View{
		Name:        "gowaku_filter_request_duration_seconds",
		Measure:     FilterRequestDurationSeconds,
		Description: "Duration of Filter Subscribe Requests",
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{ErrorType},
	}
	FilterHandleMessageDurationView = &view.View{
		Name:        "gowaku_filter_handle_msessageduration_seconds",
		Measure:     FilterHandleMessageDurationSeconds,
		Description: "Duration to Push Message to Filter Subscribers",
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{ErrorType},
	}

	LightpushMessagesView = &view.View{
		Name:        "gowaku_lightpush_messages",
		Measure:     StoredMessages,
		Description: "The distribution of the lightpush protocol messages",
		Aggregation: view.LastValue(),
		TagKeys:     []tag.Key{KeyType},
	}
	LightpushErrorTypesView = &view.View{
		Name:        "gowaku_lightpush_errors",
		Measure:     LightpushErrors,
		Description: "The distribution of the lightpush protocol errors",
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{ErrorType},
	}
	VersionView = &view.View{
		Name:        "gowaku_version",
		Measure:     WakuVersion,
		Description: "The gowaku version",
		Aggregation: view.LastValue(),
		TagKeys:     []tag.Key{GitVersion},
	}
	DnsDiscoveryNodesView = &view.View{
		Name:        "gowaku_dnsdisc_discovered",
		Measure:     DnsDiscoveryNodes,
		Description: "The number of nodes discovered via DNS discovery",
		Aggregation: view.Count(),
	}
	DnsDiscoveryErrorTypesView = &view.View{
		Name:        "gowaku_dnsdisc_errors",
		Measure:     DnsDiscoveryErrors,
		Description: "The distribution of the dns discovery protocol errors",
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{ErrorType},
	}
)

func recordWithTags(ctx context.Context, tagKey tag.Key, tagType string, ms stats.Measurement) {
	if err := stats.RecordWithTags(ctx, []tag.Mutator{tag.Insert(tagKey, tagType)}, ms); err != nil {
		utils.Logger().Error("failed to record with tags", zap.Error(err))
	}
}

func RecordLightpushMessage(ctx context.Context, tagType string) {
	if err := stats.RecordWithTags(ctx, []tag.Mutator{tag.Insert(KeyType, tagType)}, LightpushMessages.M(1)); err != nil {
		utils.Logger().Error("failed to record with tags", zap.Error(err))
	}
}

func RecordLightpushError(ctx context.Context, tagType string) {
	recordWithTags(ctx, ErrorType, tagType, LightpushErrors.M(1))
}

func RecordLegacyFilterError(ctx context.Context, tagType string) {
	recordWithTags(ctx, ErrorType, tagType, LegacyFilterErrors.M(1))
}

func RecordFilterError(ctx context.Context, tagType string) {
	recordWithTags(ctx, ErrorType, tagType, FilterErrors.M(1))
}

func RecordFilterRequest(ctx context.Context, tagType string, duration time.Duration) {
	if err := stats.RecordWithTags(ctx, []tag.Mutator{tag.Insert(KeyType, tagType)}, FilterRequests.M(1)); err != nil {
		utils.Logger().Error("failed to record with tags", zap.Error(err))
	}
	FilterRequestDurationSeconds.M(int64(duration.Seconds()))
}

func RecordFilterMessage(ctx context.Context, tagType string, len int) {
	if err := stats.RecordWithTags(ctx, []tag.Mutator{tag.Insert(KeyType, tagType)}, FilterMessages.M(int64(len))); err != nil {
		utils.Logger().Error("failed to record with tags", zap.Error(err))
	}
}

func RecordLegacyFilterMessage(ctx context.Context, tagType string, len int) {
	if err := stats.RecordWithTags(ctx, []tag.Mutator{tag.Insert(KeyType, tagType)}, LegacyFilterMessages.M(int64(len))); err != nil {
		utils.Logger().Error("failed to record with tags", zap.Error(err))
	}
}

func RecordPeerExchangeError(ctx context.Context, tagType string) {
	recordWithTags(ctx, ErrorType, tagType, PeerExchangeError.M(1))
}

func RecordDnsDiscoveryError(ctx context.Context, tagType string) {
	recordWithTags(ctx, ErrorType, tagType, DnsDiscoveryErrors.M(1))
}

func RecordStoreMessage(ctx context.Context, tagType string, len int) {
	if err := stats.RecordWithTags(ctx, []tag.Mutator{tag.Insert(KeyType, tagType)}, StoredMessages.M(int64(len))); err != nil {
		utils.Logger().Error("failed to record with tags", zap.Error(err))
	}
}

func RecordStoreQuery(ctx context.Context) {
	stats.Record(ctx, StoreQueries.M(1))
}

func RecordStoreError(ctx context.Context, tagType string) {
	recordWithTags(ctx, ErrorType, tagType, StoreErrors.M(1))
}

func RecordVersion(ctx context.Context, version string, commit string) {
	v := fmt.Sprintf("%s-%s", version, commit)
	recordWithTags(ctx, GitVersion, v, WakuVersion.M(1))
}
