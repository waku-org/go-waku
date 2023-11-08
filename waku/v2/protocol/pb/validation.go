package pb

import (
	"errors"
)

const MaxMetaAttrLength = 64

var (
	errMissingPayload      = errors.New("missing Payload field")
	errMissingContentTopic = errors.New("missing ContentTopic field")
	errInvalidMetaLength   = errors.New("invalid length for Meta field")
)

func (msg *WakuMessage) Validate() error {
	if len(msg.Payload) == 0 {
		return errMissingPayload
	}

	if msg.ContentTopic == "" {
		return errMissingContentTopic
	}

	if len(msg.Meta) > MaxMetaAttrLength {
		return errInvalidMetaLength
	}

	return nil
}
