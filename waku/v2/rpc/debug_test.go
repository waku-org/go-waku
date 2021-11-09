package rpc

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetV1Info(t *testing.T) {
	var reply InfoReply

	request, err := http.NewRequest(http.MethodPost, "url", bytes.NewReader([]byte("")))
	require.NoError(t, err)

	d := &DebugService{nil}

	err = d.GetV1Info(request, &InfoArgs{}, &reply)
	require.NoError(t, err)
	require.Equal(t, "2.0", reply.Version)
}
