package rpc

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/rpc/v2"
	"github.com/status-im/go-waku/waku/v2/node"
	"go.uber.org/zap"
)

type WakuRpc struct {
	node   *node.WakuNode
	server *http.Server

	log *zap.Logger

	relayService   *RelayService
	filterService  *FilterService
	privateService *PrivateService
	adminService   *AdminService
}

func NewWakuRpc(node *node.WakuNode, address string, port int, enableAdmin bool, enablePrivate bool, cacheCapacity int, log *zap.Logger) *WakuRpc {
	wrpc := new(WakuRpc)
	wrpc.log = log.Named("rpc")

	s := rpc.NewServer()
	s.RegisterCodec(NewSnakeCaseCodec(), "application/json")
	s.RegisterCodec(NewSnakeCaseCodec(), "application/json;charset=UTF-8")

	mux := mux.NewRouter()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t := time.Now()
		s.ServeHTTP(w, r)
		wrpc.log.Info("served request", zap.String("path", r.URL.Path), zap.Duration("duration", time.Since(t)))
	})

	debugService := NewDebugService(node)
	err := s.RegisterService(debugService, "Debug")
	if err != nil {
		wrpc.log.Error("registering debug service", zap.Error(err))
	}

	relayService := NewRelayService(node, cacheCapacity, log)
	err = s.RegisterService(relayService, "Relay")
	if err != nil {
		wrpc.log.Error("registering relay service", zap.Error(err))
	}

	err = s.RegisterService(&StoreService{node, log}, "Store")
	if err != nil {
		wrpc.log.Error("registering store service", zap.Error(err))
	}

	if enableAdmin {
		adminService := &AdminService{node, log.Named("admin")}
		err = s.RegisterService(adminService, "Admin")
		if err != nil {
			wrpc.log.Error("registering admin service", zap.Error(err))
		}
		wrpc.adminService = adminService
	}

	filterService := NewFilterService(node, cacheCapacity, log)
	err = s.RegisterService(filterService, "Filter")
	if err != nil {
		wrpc.log.Error("registering filter service", zap.Error(err))
	}

	if enablePrivate {
		privateService := NewPrivateService(node, log)
		err = s.RegisterService(privateService, "Private")
		if err != nil {
			wrpc.log.Error("registering private service", zap.Error(err))
		}
		wrpc.privateService = privateService
	}

	listenAddr := fmt.Sprintf("%s:%d", address, port)

	server := &http.Server{
		Addr:    listenAddr,
		Handler: mux,
	}

	server.RegisterOnShutdown(func() {
		filterService.Stop()
		relayService.Stop()
		if wrpc.privateService != nil {
			wrpc.privateService.Stop()
		}
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
	if r.privateService != nil {
		go r.privateService.Start()
	}
	go func() {
		_ = r.server.ListenAndServe()
	}()
	r.log.Info("server started", zap.String("addr", r.server.Addr))
}

func (r *WakuRpc) Stop(ctx context.Context) error {
	r.log.Info("shutting down server")
	return r.server.Shutdown(ctx)
}
