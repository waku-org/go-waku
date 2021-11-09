package rpc

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func makeRequest(t *testing.T) *http.Request {
	request, err := http.NewRequest(http.MethodPost, "url", bytes.NewReader([]byte("")))
	require.NoError(t, err)
	return request
}
