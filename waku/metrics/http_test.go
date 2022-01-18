package metrics

import (
	"context"
	"testing"
	"time"

	logging "github.com/ipfs/go-log"
	"github.com/stretchr/testify/require"
)

var log = logging.Logger("test")

func TestStartAndStopMetricsServer(t *testing.T) {
	server := NewMetricsServer("0.0.0.0", 9876, &log.SugaredLogger)

	go func() {
		time.Sleep(1 * time.Second)
		err := server.Stop(context.Background())
		require.NoError(t, err)
	}()

	server.Start()
}
