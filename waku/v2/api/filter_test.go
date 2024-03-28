package api

import (
	"context"
	"sync"
	"testing"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/stretchr/testify/suite"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/filter"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/protocol/subscription"
	"go.uber.org/zap"
)

func TestFilterApiSuite(t *testing.T) {
	suite.Run(t, new(FilterApiTestSuite))
}

const defaultTestPubSubTopic = "/waku/2/go/filter/test"
const defaultTestContentTopic = "/test/10/my-app"

type FilterApiTestSuite struct {
	suite.Suite

	testTopic        string
	testContentTopic string
	ctx              context.Context
	ctxCancel        context.CancelFunc
	lightNode        *filter.WakuFilterLightNode
	lightNodeHost    host.Host
	relayNode        *relay.WakuRelay
	relaySub         *relay.Subscription
	fullNode         *filter.WakuFilterFullNode
	fullNodeHost     host.Host
	wg               *sync.WaitGroup
	contentFilter    protocol.ContentFilter
	subDetails       []*subscription.SubscriptionDetails
	log              *zap.Logger
}

type WakuMsg struct {
	pubSubTopic  string
	contentTopic string
	payload      string
}
