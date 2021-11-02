package rpc

import (
	"net/http"
	"strings"

	"github.com/gorilla/rpc/v2"
	"github.com/gorilla/rpc/v2/json"
)

// SnakeCaseCodec creates a CodecRequest to process each request.
type SnakeCaseCodec struct {
}

// NewSnakeCaseCodec returns a new SnakeCaseCodec.
func NewSnakeCaseCodec() *SnakeCaseCodec {
	return &SnakeCaseCodec{}
}

// NewRequest returns a new CodecRequest of type SnakeCaseCodecRequest.
func (c *SnakeCaseCodec) NewRequest(r *http.Request) rpc.CodecRequest {
	outerCR := &SnakeCaseCodecRequest{} // Our custom CR
	jsonC := json.NewCodec()            // json Codec to create json CR
	innerCR := jsonC.NewRequest(r)      // create the json CR, sort of.

	// NOTE - innerCR is of the interface type rpc.CodecRequest.
	// Because innerCR is of the rpc.CR interface type, we need a
	// type assertion in order to assign it to our struct field's type.
	// We defined the source of the interface implementation here, so
	// we can be confident that innerCR will be of the correct underlying type
	outerCR.CodecRequest = innerCR.(*json.CodecRequest)
	return outerCR
}

// SnakeCaseCodecRequest decodes and encodes a single request. SnakeCaseCodecRequest
// implements gorilla/rpc.CodecRequest interface primarily by embedding
// the CodecRequest from gorilla/rpc/json. By selectively adding
// CodecRequest methods to SnakeCaseCodecRequest, we can modify that behaviour
// while maintaining all the other remaining CodecRequest methods from
// gorilla's rpc/json implementation
type SnakeCaseCodecRequest struct {
	*json.CodecRequest
}

// Method returns the decoded method as a string of the form "Service.Method"
// after checking for, and correcting a lowercase method name
// By being of lower depth in the struct , Method will replace the implementation
// of Method() on the embedded CodecRequest. Because the request data is part
// of the embedded json.CodecRequest, and unexported, we have to get the
// requested method name via the embedded CR's own method Method().
// Essentially, this just intercepts the return value from the embedded
// gorilla/rpc/json.CodecRequest.Method(), checks/modifies it, and passes it
// on to the calling rpc server.
func (c *SnakeCaseCodecRequest) Method() (string, error) {
	m, err := c.CodecRequest.Method()
	return snakeCaseToCamelCase(m), err
}

func snakeCaseToCamelCase(inputUnderScoreStr string) (camelCase string) {
	isToUpper := false
	for k, v := range inputUnderScoreStr {
		if k == 0 {
			camelCase = strings.ToUpper(string(inputUnderScoreStr[0]))
		} else {
			if isToUpper {
				camelCase += strings.ToUpper(string(v))
				isToUpper = false
			} else {
				if v == '_' {
					isToUpper = true
				} else if v == '.' {
					isToUpper = true
					camelCase += string(v)
				} else {
					camelCase += string(v)
				}
			}
		}
	}
	return

}
