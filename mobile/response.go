package gowaku

import "encoding/json"

type jsonResponse struct {
	Error  *string     `json:"error,omitempty"`
	Result interface{} `json:"result"`
}

func prepareJSONResponse(result interface{}, err error) string {

	if err != nil {
		errStr := err.Error()
		errResponse := jsonResponse{
			Error: &errStr,
		}
		response, _ := json.Marshal(&errResponse)
		return string(response)
	}

	data, err := json.Marshal(jsonResponse{Result: result})
	if err != nil {
		return prepareJSONResponse(nil, err)
	}
	return string(data)
}

func makeJSONResponse(err error) string {
	var errString *string = nil
	if err != nil {
		errStr := err.Error()
		errString = &errStr
	}

	out := jsonResponse{
		Error: errString,
	}
	outBytes, _ := json.Marshal(out)

	return string(outBytes)
}
