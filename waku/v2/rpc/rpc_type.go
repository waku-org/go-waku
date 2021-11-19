package rpc

import "github.com/status-im/go-waku/waku/v2/protocol/pb"

type SuccessReply struct {
	Success bool   `json:"success,omitempty"`
	Error   string `json:"error,omitempty"`
}

type Empty struct {
}

type MessagesReply struct {
	Messages []*pb.WakuMessage `json:"messages,omitempty"`
}
