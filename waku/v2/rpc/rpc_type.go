package rpc

import (
	"encoding/base64"
	"strings"
)

type SuccessReply = bool

type Empty struct {
}

type MessagesReply = []*RPCWakuMessage

type Base64URLByte []byte

func (u Base64URLByte) MarshalJSON() ([]byte, error) {
	base64Value := base64.URLEncoding.EncodeToString(u)
	return []byte("\"" + base64Value + "\""), nil
}

func (h *Base64URLByte) UnmarshalText(b []byte) error {
	inputValue := ""
	if b != nil {
		inputValue = string(b)
	}

	enc := base64.StdEncoding
	if strings.ContainsAny(inputValue, "-_") {
		enc = base64.URLEncoding
	}
	if len(inputValue)%4 != 0 {
		enc = enc.WithPadding(base64.NoPadding)
	}

	decodedBytes, err := enc.DecodeString(inputValue)
	if err != nil {
		return err
	}

	*h = decodedBytes

	return nil
}
