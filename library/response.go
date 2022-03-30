package main

import "C"
import (
	"encoding/json"
)

type JSONResponse struct {
	Error  *string     `json:"error,omitempty"`
	Result interface{} `json:"result"`
}

func prepareJSONResponse(result interface{}, err error) *C.char {

	if err != nil {
		errStr := err.Error()
		errResponse := JSONResponse{
			Error: &errStr,
		}
		response, _ := json.Marshal(&errResponse)
		return C.CString(string(response))
	}

	data, err := json.Marshal(JSONResponse{Result: result})
	if err != nil {
		return prepareJSONResponse(nil, err)
	}
	return C.CString(string(data))
}

func makeJSONResponse(err error) *C.char {
	var errString *string = nil
	if err != nil {
		errStr := err.Error()
		errString = &errStr
	}

	out := JSONResponse{
		Error: errString,
	}
	outBytes, _ := json.Marshal(out)

	return C.CString(string(outBytes))
}
