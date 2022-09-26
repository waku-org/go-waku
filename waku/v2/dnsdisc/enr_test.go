package dnsdisc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestRetrieveNodes uses a live connection, so it could be
// flaky, it should though pay for itself and should be fairly stable
func TestRetrieveNodes(t *testing.T) {
	url := "enrtree://AOGECG2SPND25EEFMAJ5WF3KSGJNSGV356DSTL2YVLLZWIV6SAYBM@test.waku.nodes.status.im"

	nodes, err := RetrieveNodes(context.Background(), url)
	require.NoError(t, err)
	require.NotEmpty(t, nodes)
}
