package metrics

import (
	"context"
	"fmt"
	"net/http"

	"contrib.go.opencensus.io/exporter/prometheus"
	logging "github.com/ipfs/go-log"
	"github.com/status-im/go-waku/waku/v2/metrics"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/plugin/runmetrics"
	"go.opencensus.io/stats/view"
)

var log = logging.Logger("metrics")

// Server runs and controls a HTTP pprof interface.
type Server struct {
	server *http.Server
}

// NewMetricsServer creates a prometheus server on a particular interface and port
func NewMetricsServer(address string, port int) *Server {
	_ = runmetrics.Enable(runmetrics.RunMetricOptions{
		EnableCPU:    true,
		EnableMemory: true,
	})

	pe, err := prometheus.NewExporter(prometheus.Options{})
	if err != nil {
		log.Fatalf("Failed to create the Prometheus stats exporter: %v", err)
	}

	view.RegisterExporter(pe)

	log.Info(fmt.Sprintf("Starting server at  %s:%d", address, port))
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
		metrics.FilterSubscriptionsView,
		metrics.StoreErrorTypesView,
		metrics.StoreMessagesView,
		metrics.PeersView,
		metrics.DialsView,
	); err != nil {
		log.Fatalf("Failed to register views: %v", err)
	}

	p := Server{
		server: &http.Server{
			Addr:    fmt.Sprintf("%s:%d", address, port),
			Handler: h,
		},
	}

	return &p
}

// Start executes the HTTP server in the background.
func (p *Server) Start() {
	log.Info("Server stopped ", p.server.ListenAndServe())
}

// Stop shuts down the prometheus server
func (p *Server) Stop(ctx context.Context) {
	err := p.server.Shutdown(ctx)
	if err != nil {
		log.Error("Server shutdown err", err)
	}
}
