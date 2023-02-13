package rpc

import "github.com/waku-org/go-waku/waku/v2/protocol/pb"

type SuccessReply = bool

type Empty struct {
}

type MessagesReply = []*pb.WakuMessage

type RelayMessagesReply = []*pb.WakuMessage
