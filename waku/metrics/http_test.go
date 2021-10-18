package metrics

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestStartAndStopMetricsServer(t *testing.T) {
	server := NewMetricsServer("0.0.0.0", 9876)

	go func() {
		time.Sleep(1 * time.Second)
		err := server.Stop(context.Background())
		require.NoError(t, err)
	}()

	server.Start()
}
