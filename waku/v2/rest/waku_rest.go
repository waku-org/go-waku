package rest

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"
	"sync"

	"github.com/gorilla/mux"
	"github.com/waku-org/go-waku/waku/v2/node"
	"go.uber.org/zap"
)

type WakuRest struct {
	node   *node.WakuNode
	server *http.Server

	log *zap.Logger

	relayService *RelayService
}

func NewWakuRest(node *node.WakuNode, address string, port int, enableAdmin bool, enablePrivate bool, enablePProf bool, relayCacheCapacity int, log *zap.Logger) *WakuRest {
	wrpc := new(WakuRest)
	wrpc.log = log.Named("rest")

	mux := mux.NewRouter()

	if enablePProf {
		mux.PathPrefix("/debug/").Handler(http.DefaultServeMux)
		mux.HandleFunc("/debug/pprof/", pprof.Index)
	}

	_ = NewDebugService(node, mux)
	relayService := NewRelayService(node, mux, relayCacheCapacity, log)

	listenAddr := fmt.Sprintf("%s:%d", address, port)

	server := &http.Server{
		Addr:    listenAddr,
		Handler: mux,
	}

	server.RegisterOnShutdown(func() {
		relayService.Stop()
	})

	wrpc.node = node
	wrpc.server = server
	wrpc.relayService = relayService

	return wrpc
}

func (r *WakuRest) Start(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	go r.relayService.Start(ctx)
	go func() {
		_ = r.server.ListenAndServe()
	}()
	r.log.Info("server started", zap.String("addr", r.server.Addr))
}

func (r *WakuRest) Stop(ctx context.Context) error {
	r.log.Info("shutting down server")
	return r.server.Shutdown(ctx)
}
