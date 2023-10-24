package rpc

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
)

func makeRequest(t *testing.T) *http.Request {
	request, err := http.NewRequest(http.MethodPost, "url", bytes.NewReader([]byte("")))
	require.NoError(t, err)
	return request
}

func TestBase64Encoding(t *testing.T) {
	input := "Hello World"

	rpcMsg, err := ProtoToRPC(&pb.WakuMessage{
		Payload: []byte(input),
	})
	require.NoError(t, err)

	jsonBytes, err := json.Marshal(rpcMsg)
	require.NoError(t, err)

	m := make(map[string]interface{})
	err = json.Unmarshal(jsonBytes, &m)
	require.NoError(t, err)
	require.Equal(t, base64.StdEncoding.EncodeToString([]byte(input)), m["payload"])

	decodedRPCMsg := new(RPCWakuMessage)
	err = json.Unmarshal(jsonBytes, decodedRPCMsg)
	require.NoError(t, err)
	require.Equal(t, input, string(decodedRPCMsg.Payload))
}
