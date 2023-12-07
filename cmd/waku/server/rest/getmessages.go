package rest

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/waku-org/go-waku/waku/persistence"
	"github.com/waku-org/go-waku/waku/persistence/sqlite"
	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-waku/waku/v2/protocol/store/pb"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"go.uber.org/zap"
)

type GetMessagesService struct {
	node *node.WakuNode
	mux  *chi.Mux

	log *zap.Logger
}

const routeGetMessagesV1 = "/store/v1/getmessages"

func NewGetMessagesService(node *node.WakuNode, m *chi.Mux) *GetMessagesService {
	g := &GetMessagesService{
		node: node,
		mux:  m,
	}
	m.Post(routeGetMessagesV1, g.queryForV1GetMessages)

	return g
}

type GetMessagesResponse string

func (g *GetMessagesService) queryForV1GetMessages(w http.ResponseWriter, req *http.Request) {

	queryCriteria := &pb.HistoryQuery{}
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(queryCriteria); err != nil {
		g.log.Error("bad request", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	db, err := sqlite.NewDB("my_store2_copy.db", utils.Logger())
	if err != nil {
		fmt.Println("newDB error: " + err.Error())
	}

	dbStore, err := persistence.NewDBStore(prometheus.DefaultRegisterer, utils.Logger(), persistence.WithDB(db), persistence.WithMigrations(sqlite.Migrations))
	if err != nil {
		fmt.Println("newDBStore error: " + err.Error())
	}

	_, msgs, err := dbStore.Query(queryCriteria)
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Printf("msgs returned: %d \n", len(msgs))

	writeResponse(w, msgs, http.StatusOK)
}
