package rpc

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/rpc/v2"
	"github.com/status-im/go-waku/waku/v2/node"
	"go.uber.org/zap"
)

type WakuRpc struct {
	node   *node.WakuNode
	server *http.Server

	log *zap.SugaredLogger

	relayService   *RelayService
	filterService  *FilterService
	privateService *PrivateService
	adminService   *AdminService
}

func NewWakuRpc(node *node.WakuNode, address string, port int, enableAdmin bool, enablePrivate bool, log *zap.SugaredLogger) *WakuRpc {
	wrpc := new(WakuRpc)
	wrpc.log = log.Named("rpc")

	s := rpc.NewServer()
	s.RegisterCodec(NewSnakeCaseCodec(), "application/json")
	s.RegisterCodec(NewSnakeCaseCodec(), "application/json;charset=UTF-8")

	err := s.RegisterService(&DebugService{node}, "Debug")
	if err != nil {
		wrpc.log.Error(err)
	}

	relayService := NewRelayService(node, log)
	err = s.RegisterService(relayService, "Relay")
	if err != nil {
		wrpc.log.Error(err)
	}

	err = s.RegisterService(&StoreService{node, log}, "Store")
	if err != nil {
		wrpc.log.Error(err)
	}

	if enableAdmin {
		adminService := &AdminService{node, log.Named("admin")}
		err = s.RegisterService(adminService, "Admin")
		if err != nil {
			wrpc.log.Error(err)
		}
		wrpc.adminService = adminService
	}

	filterService := NewFilterService(node, log)
	err = s.RegisterService(filterService, "Filter")
	if err != nil {
		wrpc.log.Error(err)
	}

	if enablePrivate {
		privateService := NewPrivateService(node, log)
		err = s.RegisterService(privateService, "Private")
		if err != nil {
			wrpc.log.Error(err)
		}
		wrpc.privateService = privateService
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/jsonrpc", func(w http.ResponseWriter, r *http.Request) {
		t := time.Now()
		s.ServeHTTP(w, r)
		wrpc.log.Infof("RPC request at %s took %s", r.URL.Path, time.Since(t))
	})

	listenAddr := fmt.Sprintf("%s:%d", address, port)

	server := &http.Server{
		Addr:    listenAddr,
		Handler: mux,
	}

	server.RegisterOnShutdown(func() {
		filterService.Stop()
		relayService.Stop()
	})

	wrpc.node = node
	wrpc.server = server
	wrpc.relayService = relayService
	wrpc.filterService = filterService

	return wrpc
}

func (r *WakuRpc) Start() {
	go r.relayService.Start()
	go r.filterService.Start()
	go func() {
		_ = r.server.ListenAndServe()
	}()
	r.log.Info("Rpc server started at ", r.server.Addr)
}

func (r *WakuRpc) Stop(ctx context.Context) error {
	r.log.Info("Shutting down rpc server")
	return r.server.Shutdown(ctx)
}
