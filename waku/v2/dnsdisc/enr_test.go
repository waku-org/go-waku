package dnsdisc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestRetrieveNodes uses a live connection, so it could be
// flaky, it should though pay for itself and should be fairly stable
func TestRetrieveNodes(t *testing.T) {
	url := "enrtree://AOFTICU2XWDULNLZGRMQS4RIZPAZEHYMV4FYHAPW563HNRAOERP7C@test.waku.nodes.status.im"

	nodes, err := RetrieveNodes(context.Background(), url)
	require.NoError(t, err)
	require.NotEmpty(t, nodes)
}
