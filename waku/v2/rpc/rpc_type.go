package rpc

type SuccessReply struct {
	Success bool   `json:"success,omitempty"`
	Error   string `json:"error,omitempty"`
}
