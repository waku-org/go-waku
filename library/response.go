package main

import "C"
import (
	"encoding/json"
)

const (
	codeUnknown int = iota
	// special codes
	codeFailedParseResponse
	// codeFailedParseParams
)

var errToCodeMap = map[error]int{
	//transactions.ErrInvalidTxSender: codeErrInvalidTxSender,
}

type jsonrpcSuccessfulResponse struct {
	Result interface{} `json:"result"`
}

type jsonrpcErrorResponse struct {
	Error jsonError `json:"error"`
}

type jsonError struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message"`
}

func prepareJSONResponse(result interface{}, err error) *C.char {
	code := codeUnknown
	if c, ok := errToCodeMap[err]; ok {
		code = c
	}

	return prepareJSONResponseWithCode(result, err, code)
}

func prepareJSONResponseWithCode(result interface{}, err error, code int) *C.char {
	if err != nil {
		errResponse := jsonrpcErrorResponse{
			Error: jsonError{Code: code, Message: err.Error()},
		}
		response, _ := json.Marshal(&errResponse)
		return C.CString(string(response))
	}

	data, err := json.Marshal(jsonrpcSuccessfulResponse{result})
	if err != nil {
		return prepareJSONResponseWithCode(nil, err, codeFailedParseResponse)
	}
	return C.CString(string(data))
}

func makeJSONResponse(err error) *C.char {
	var errString *string = nil
	if err != nil {
		errStr := err.Error()
		errString = &errStr
	}

	out := APIResponse{
		Error: errString,
	}
	outBytes, _ := json.Marshal(out)

	return C.CString(string(outBytes))
}
