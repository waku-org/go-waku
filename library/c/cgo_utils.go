package main

import (
	"encoding/json"
)

/*
#include <stdlib.h>
*/
import "C"

func execOkCB(value any) *C.char {
	return C.CString(prepareJSONResponse(value, nil))
}

func execErrCB(err error) *C.char {
	return C.CString(makeJSONResponse(err))
}

type jsonResponseError struct {
	Error *string `json:"error"`
}

type jsonResponseSuccess struct {
	Result interface{} `json:"result"`
}

func prepareJSONResponse(result interface{}, err error) string {

	if err != nil {
		errStr := err.Error()
		errResponse := jsonResponseError{
			Error: &errStr,
		}
		response, _ := json.Marshal(&errResponse)
		return string(response)
	}

	data, err := json.Marshal(jsonResponseSuccess{Result: result})
	if err != nil {
		return prepareJSONResponse(nil, err)
	}
	return string(data)
}

func makeJSONResponse(err error) string {
	if err != nil {
		errStr := err.Error()
		outBytes, _ := json.Marshal(jsonResponseError{Error: &errStr})
		return string(outBytes)
	}

	out := jsonResponseSuccess{
		Result: true,
	}
	outBytes, _ := json.Marshal(out)

	return string(outBytes)
}
