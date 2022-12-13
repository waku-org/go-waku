package rpc

import (
	"net/http"

	"github.com/waku-org/go-waku/waku/v2/protocol"
)

type WakuV2RPC interface {
	Start()
	Stop()
	addEnvelope(envelope *protocol.Envelope)
	PostV1Message(req *http.Request, args interface{}, reply *SuccessReply) error
	GetV1Messages(req *http.Request, args interface{}, reply interface{}) error
	PostV1Subscription(req *http.Request, args interface{}, reply *SuccessReply) error
	DeleteV1Subscription(req *http.Request, args interface{}, reply *SuccessReply) error
}
