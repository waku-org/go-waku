package utils

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTtlMap(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ttlMap := NewTtlMap[string, bool](ctx, 3)
	ttlMap.Put("a", true)
	ttlMap.Put("b", false)

	v, ok := ttlMap.Get("a")
	require.Equal(t, v, true)
	require.Equal(t, ok, true)

	v, ok = ttlMap.Get("b")
	require.Equal(t, v, false)
	require.Equal(t, ok, true)

	time.Sleep(5 * time.Second)

	require.Equal(t, ttlMap.Len(), 0)
	ttlMap.Put("c", true)
	cancel()
	time.Sleep(5 * time.Second)
	require.Equal(t, ttlMap.Len(), 1)

}
