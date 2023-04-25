package metrics

import (
	"context"
	"fmt"
	"net/http"

	"contrib.go.opencensus.io/exporter/prometheus"
	"github.com/waku-org/go-waku/waku/v2/metrics"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/plugin/runmetrics"
	"go.opencensus.io/stats/view"
	"go.uber.org/zap"
)

// Server runs and controls a HTTP pprof interface.
type Server struct {
	server *http.Server
	log    *zap.Logger
}

// NewMetricsServer creates a prometheus server on a particular interface and port
func NewMetricsServer(address string, port int, log *zap.Logger) *Server {
	p := Server{
		log: log.Named("metrics"),
	}

	_ = runmetrics.Enable(runmetrics.RunMetricOptions{
		EnableCPU:    true,
		EnableMemory: true,
	})

	pe, err := prometheus.NewExporter(prometheus.Options{})
	if err != nil {
		p.log.Fatal("creating Prometheus stats exporter", zap.Error(err))
	}

	view.RegisterExporter(pe)

	p.log.Info("starting server", zap.String("address", address), zap.Int("port", port))
	mux := http.NewServeMux()
	mux.Handle("/metrics", pe)

	// Healthcheck
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "OK")
	})

	h := &ochttp.Handler{Handler: mux}

	// Register the views
	if err := view.Register(
		metrics.MessageView,
		metrics.MessageSizeView,
		metrics.LegacyFilterErrorTypesView,
		metrics.LegacyFilterMessagesView,
		metrics.LegacyFilterSubscribersView,
		metrics.LegacyFilterSubscriptionsView,
		metrics.FilterSubscriptionsView,
		metrics.FilterErrorTypesView,
		metrics.FilterHandleMessageDurationView,
		metrics.FilterMessagesView,
		metrics.FilterRequestDurationView,
		metrics.FilterRequestsView,
		metrics.LightpushMessagesView,
		metrics.LightpushErrorTypesView,
		metrics.DnsDiscoveryNodesView,
		metrics.DnsDiscoveryErrorTypesView,
		metrics.DiscV5ErrorTypesView,
		metrics.StoreErrorTypesView,
		metrics.StoreQueriesView,
		metrics.ArchiveErrorTypesView,
		metrics.ArchiveInsertDurationView,
		metrics.ArchiveMessagesView,
		metrics.ArchiveQueryDurationView,
		metrics.StoreErrorTypesView,
		metrics.StoreQueriesView,
		metrics.PeersView,
		metrics.DialsView,
		metrics.VersionView,
	); err != nil {
		p.log.Fatal("registering views", zap.Error(err))
	}

	p.server = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", address, port),
		Handler: h,
	}

	return &p
}

// Start executes the HTTP server in the background.
func (p *Server) Start() {
	p.log.Info("server stopped ", zap.Error(p.server.ListenAndServe()))
}

// Stop shuts down the prometheus server
func (p *Server) Stop(ctx context.Context) error {
	err := p.server.Shutdown(ctx)
	if err != nil {
		p.log.Error("stopping server", zap.Error(err))
		return err
	}

	return nil
}
