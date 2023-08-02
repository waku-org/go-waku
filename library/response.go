package library

import "encoding/json"

func MarshalJSON(result interface{}) (string, error) {
	data, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
