package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGetUnixEpochFrom(t *testing.T) {
	loc := time.UTC
	timestamp := GetUnixEpochFrom(time.Date(2019, 1, 1, 0, 0, 0, 0, loc))

	require.Equal(t, float64(1546300800), timestamp)
}
