package metrics

import (
	"context"
	"fmt"
	"net/http"

	"contrib.go.opencensus.io/exporter/prometheus"
	logging "github.com/ipfs/go-log"
	"github.com/status-im/go-waku/waku/v2/metrics"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/stats/view"
)

var log = logging.Logger("metrics")

// Server runs and controls a HTTP pprof interface.
type Server struct {
	server *http.Server
}

func NewMetricsServer(address string, port int) *Server {
	pe, err := prometheus.NewExporter(prometheus.Options{
		Namespace: "wakunode",
	})
	if err != nil {
		log.Fatalf("Failed to create the Prometheus stats exporter: %v", err)
	}

	view.RegisterExporter(pe)

	log.Info(fmt.Sprintf("Starting server at  %s:%d", address, port))
	mux := http.NewServeMux()
	mux.Handle("/metrics", pe)

	h := &ochttp.Handler{Handler: mux}

	// Register the views
	if err := view.Register(
		metrics.MessageTypeView,
		metrics.FilterSubscriptionsView,
		metrics.StoreErrorTypesView,
		metrics.StoreMessageTypeView,
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

// Listen starts the HTTP server in the background.
func (p *Server) Start() {
	log.Info("Server stopped ", p.server.ListenAndServe())
}

func (p *Server) Stop(ctx context.Context) {
	err := p.server.Shutdown(ctx)
	if err != nil {
		log.Error("Server shutdown err", err)
	}
}
