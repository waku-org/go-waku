package publish

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

func TestRlnRateLimit(t *testing.T) {
	r := NewRlnRateLimiter(3, 5*time.Second)
	l := utils.Logger()

	for i := 0; i < 3; i++ {
		require.NoError(t, r.Check(context.Background(), l))
	}
	require.ErrorIs(t, r.Check(context.Background(), l), ErrRateLimited)

	time.Sleep(6 * time.Second)
	for i := 0; i < 3; i++ {
		require.NoError(t, r.Check(context.Background(), l))
	}
	require.ErrorIs(t, r.Check(context.Background(), l), ErrRateLimited)
}
