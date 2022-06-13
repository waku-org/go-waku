package rpc

type SuccessReply struct {
	Success bool   `json:"success,omitempty"`
	Error   string `json:"error,omitempty"`
}

type Empty struct {
}

type MessagesReply []*RPCWakuMessage

type RelayMessagesReply []*RPCWakuRelayMessage
