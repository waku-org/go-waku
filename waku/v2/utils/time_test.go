package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGetUnixEpochFrom(t *testing.T) {
	loc := time.UTC
	timeFn := func() time.Time {
		return time.Date(2019, 1, 1, 0, 0, 0, 0, loc)
	}
	timestamp := GetUnixEpochFrom(timeFn)

	require.Equal(t, float64(1546300800), timestamp)
}
