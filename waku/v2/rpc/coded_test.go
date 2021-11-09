package rpc

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConvertWakuMethod(t *testing.T) {
	res := toWakuMethod("get_waku_v2_debug_v1_info")
	require.Equal(t, "Debug.GetV1Info", res)

	res = toWakuMethod("post_waku_v2_relay_v1_message")
	require.Equal(t, "Relay.PostV1Message", res)

	res = toWakuMethod("delete_waku_v2_relay_v1_subscriptions")
	require.Equal(t, "Relay.DeleteV1Subscriptions", res)
}
