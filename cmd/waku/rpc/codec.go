package rpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/gorilla/rpc/v2"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Based on github.com/gorilla/rpc/v2/json which is governed by a BSD-style license

var null = json.RawMessage([]byte("null"))

// An Error is a wrapper for a JSON interface value. It can be used by either
// a service's handler func to write more complex JSON data to an error field
// of a server's response, or by a client to read it.
type Error struct {
	Data interface{}
}

func (e *Error) Error() string {
	return fmt.Sprintf("%v", e.Data)
}

// ----------------------------------------------------------------------------
// Request and Response
// ----------------------------------------------------------------------------

// serverRequest represents a JSON-RPC request received by the server.
type serverRequest struct {
	// A String containing the name of the method to be invoked.
	Method string `json:"method"`
	// An Array of objects to pass as arguments to the method.
	Params *json.RawMessage `json:"params"`
	// The request id. This can be of any type. It is used to match the
	// response with the request that it is replying to.
	Id *json.RawMessage `json:"id"`
}

// serverResponse represents a JSON-RPC response returned by the server.
type serverResponse struct {
	// The Object that was returned by the invoked method. This must be null
	// in case there was an error invoking the method.
	Result interface{} `json:"result"`
	// An Error object if there was an error invoking the method. It must be
	// null if there was no error.
	Error interface{} `json:"error"`
	// This must be the same id as the request it is responding to.
	Id *json.RawMessage `json:"id"`
}

// ----------------------------------------------------------------------------
// Codec
// ----------------------------------------------------------------------------

// NewCodec returns a new SnakeCaseCodec Codec.
func NewSnakeCaseCodec() *SnakeCaseCodec {
	return &SnakeCaseCodec{}
}

// SnakeCaseCodec creates a CodecRequest to process each request.
type SnakeCaseCodec struct {
}

// NewRequest returns a CodecRequest.
func (c *SnakeCaseCodec) NewRequest(r *http.Request) rpc.CodecRequest {
	return newCodecRequest(r)
}

// ----------------------------------------------------------------------------
// CodecRequest
// ----------------------------------------------------------------------------

// newCodecRequest returns a new CodecRequest.
func newCodecRequest(r *http.Request) rpc.CodecRequest {
	// Decode the request body and check if RPC method is valid.
	req := new(serverRequest)
	err := json.NewDecoder(r.Body).Decode(req)
	r.Body.Close()
	return &CodecRequest{request: req, err: err}
}

// CodecRequest decodes and encodes a single request.
type CodecRequest struct {
	request *serverRequest
	err     error
}

// Method returns the RPC method for the current request.
//
// The method uses a dotted notation as in "Service.Method".
func (c *CodecRequest) Method() (string, error) {
	if c.err == nil {
		return toWakuMethod(c.request.Method), nil
	}
	return "", c.err
}

// toWakuMethod transform get_waku_v2_debug_v1_info to Debug.GetV1Info
func toWakuMethod(input string) string {
	typ := "get"
	if strings.HasPrefix(input, "post") {
		typ = "post"
	} else if strings.HasPrefix(input, "delete") {
		typ = "delete"
	}

	base := typ + "_waku_v2_"
	cleanedInput := strings.Replace(input, base, "", 1)
	splited := strings.Split(cleanedInput, "_")

	c := cases.Title(language.AmericanEnglish)

	method := c.String(typ)
	for _, val := range splited[1:] {
		method = method + c.String(val)
	}

	return c.String(splited[0]) + "." + method
}

// ReadRequest fills the request object for the RPC method.
func (c *CodecRequest) ReadRequest(args interface{}) error {
	if c.err == nil {
		if c.request.Params != nil {
			// JSON params is array value. RPC params is struct.
			// Attempt to unmarshal into array containing the request struct.
			params := [1]interface{}{args}
			err := json.Unmarshal(*c.request.Params, &params)
			if err != nil {
				// This failed so we might have received an array of parameters
				// instead of a object
				argsValueOf := reflect.Indirect(reflect.ValueOf(args))
				if argsValueOf.Kind() == reflect.Struct {
					var params []interface{}
					for i := 0; i < argsValueOf.NumField(); i++ {
						params = append(params, argsValueOf.Field(i).Addr().Interface())
					}
					c.err = json.Unmarshal(*c.request.Params, &params)
				} else {
					// Unknown field type...
					c.err = err
				}
			}

		} else {
			c.err = errors.New("rpc: method request ill-formed: missing params field")
		}
	}
	return c.err
}

// WriteResponse encodes the response and writes it to the ResponseWriter.
func (c *CodecRequest) WriteResponse(w http.ResponseWriter, reply interface{}) {
	if c.request.Id != nil {
		// Id is null for notifications and they don't have a response.
		res := &serverResponse{
			Result: reply,
			Error:  &null,
			Id:     c.request.Id,
		}
		c.writeServerResponse(w, 200, res)
	}
}

func (c *CodecRequest) WriteError(w http.ResponseWriter, _ int, err error) {
	res := &serverResponse{
		Result: &null,
		Id:     c.request.Id,
	}
	if jsonErr, ok := err.(*Error); ok {
		res.Error = jsonErr.Data
	} else {
		res.Error = err.Error()
	}
	c.writeServerResponse(w, 400, res)
}

func (c *CodecRequest) writeServerResponse(w http.ResponseWriter, status int, res *serverResponse) {
	b, err := json.Marshal(res)
	if err == nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(status)
		_, _ = w.Write(b)
	} else {
		// Not sure in which case will this happen. But seems harmless.
		rpc.WriteError(w, status, err.Error())
	}
}
