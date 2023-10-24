package pb

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
)

func TestValidateRequest(t *testing.T) {
	request := PushRPC{}
	require.ErrorIs(t, request.ValidateRequest(), errMissingRequestID)
	request.RequestId = "test"
	require.ErrorIs(t, request.ValidateRequest(), errMissingQuery)
	request.Query = &PushRequest{}
	require.ErrorIs(t, request.ValidateRequest(), errMissingPubsubTopic)
	request.Query.PubsubTopic = "test"
	require.ErrorIs(t, request.ValidateRequest(), errMissingMessage)
	request.Query.Message = &pb.WakuMessage{
		Payload:      []byte{1, 2, 3},
		ContentTopic: "test",
	}
	require.NoError(t, request.ValidateRequest())
}

func TestValidateResponse(t *testing.T) {
	response := PushRPC{}
	require.ErrorIs(t, response.ValidateResponse("test"), errMissingRequestID)
	response.RequestId = "test1"
	require.ErrorIs(t, response.ValidateResponse("test"), errRequestIDMismatch)
	response.RequestId = "test"
	require.ErrorIs(t, response.ValidateResponse("test"), errMissingResponse)
	response.Response = &PushResponse{}
	require.NoError(t, response.ValidateResponse("test"))
}
