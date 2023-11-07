package rest

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/waku-org/go-waku/waku/v2/node"
	"go.uber.org/zap"
)

type WakuRest struct {
	node   *node.WakuNode
	server *http.Server

	log *zap.Logger

	relayService  *RelayService
	filterService *FilterService
}

func NewWakuRest(node *node.WakuNode, address string, port int, enablePProf bool, enableAdmin bool, relayCacheCapacity, filterCacheCapacity int, log *zap.Logger) *WakuRest {
	wrpc := new(WakuRest)
	wrpc.log = log.Named("rest")

	mux := chi.NewRouter()
	mux.Use(middleware.Logger)
	mux.Use(middleware.NoCache)

	if enablePProf {
		mux.Mount("/debug", middleware.Profiler())
	}

	_ = NewDebugService(node, mux)
	_ = NewHealthService(node, mux)
	_ = NewStoreService(node, mux)
	_ = NewLightpushService(node, mux, log)

	listenAddr := fmt.Sprintf("%s:%d", address, port)

	server := &http.Server{
		Addr:    listenAddr,
		Handler: mux,
	}

	wrpc.node = node
	wrpc.server = server

	if node.Relay() != nil {
		relayService := NewRelayService(node, mux, relayCacheCapacity, log)
		wrpc.relayService = relayService
	}

	if enableAdmin {
		_ = NewAdminService(node, mux, wrpc.log)
	}

	if node.FilterLightnode() != nil {
		filterService := NewFilterService(node, mux, filterCacheCapacity, log)
		server.RegisterOnShutdown(func() {
			filterService.Stop()
		})
		wrpc.filterService = filterService
	}

	return wrpc
}

func (r *WakuRest) Start(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	if r.node.FilterLightnode() != nil {
		go r.filterService.Start(ctx)
	}

	go func() {
		_ = r.server.ListenAndServe()
	}()
	r.log.Info("server started", zap.String("addr", r.server.Addr))
}

func (r *WakuRest) Stop(ctx context.Context) error {
	r.log.Info("shutting down server")
	return r.server.Shutdown(ctx)
}
