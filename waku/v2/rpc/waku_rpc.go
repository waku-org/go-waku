package rpc

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/rpc/v2"
	logging "github.com/ipfs/go-log"
	"github.com/status-im/go-waku/waku/v2/node"
)

var log = logging.Logger("wakurpc")

type WakuRpc struct {
	node   *node.WakuNode
	server *http.Server

	relayService   *RelayService
	filterService  *FilterService
	privateService *PrivateService
}

func NewWakuRpc(node *node.WakuNode, address string, port int) *WakuRpc {
	s := rpc.NewServer()
	s.RegisterCodec(NewSnakeCaseCodec(), "application/json")
	s.RegisterCodec(NewSnakeCaseCodec(), "application/json;charset=UTF-8")

	err := s.RegisterService(&DebugService{node}, "Debug")
	if err != nil {
		log.Error(err)
	}

	relayService := NewRelayService(node)
	err = s.RegisterService(relayService, "Relay")
	if err != nil {
		log.Error(err)
	}

	err = s.RegisterService(&StoreService{node}, "Store")
	if err != nil {
		log.Error(err)
	}

	err = s.RegisterService(&AdminService{node}, "Admin")
	if err != nil {
		log.Error(err)
	}

	filterService := NewFilterService(node)
	err = s.RegisterService(filterService, "Filter")
	if err != nil {
		log.Error(err)
	}

	privateService := NewPrivateService(node)
	err = s.RegisterService(privateService, "Private")
	if err != nil {
		log.Error(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/jsonrpc", func(w http.ResponseWriter, r *http.Request) {
		t := time.Now()
		s.ServeHTTP(w, r)
		log.Infof("RPC request at %s took %s", r.URL.Path, time.Since(t))
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

	return &WakuRpc{
		node:           node,
		server:         server,
		relayService:   relayService,
		filterService:  filterService,
		privateService: privateService,
	}
}

func (r *WakuRpc) Start() {
	go r.relayService.Start()
	go r.filterService.Start()
	go func() {
		_ = r.server.ListenAndServe()
	}()
	log.Info("Rpc server started at ", r.server.Addr)
}

func (r *WakuRpc) Stop(ctx context.Context) error {
	log.Info("Shutting down rpc server")
	return r.server.Shutdown(ctx)
}
