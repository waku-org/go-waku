package main

/*
#include <stdlib.h>
#include <stddef.h>
*/
import "C"
import (
	"encoding/base64"
	"unsafe"
)

//export waku_utils_base64_decode
// Decode a base64 string (useful for reading the payload from waku messages)
func waku_utils_base64_decode(data *C.char) *C.char {
	b, err := base64.StdEncoding.DecodeString(C.GoString(data))
	if err != nil {
		return makeJSONResponse(err)
	}

	return prepareJSONResponse(string(b), nil)
}

//export waku_utils_base64_encode
// Encode data to base64 (useful for creating the payload of a waku message in the
// format understood by waku_relay_publish)
func waku_utils_base64_encode(data *C.char) *C.char {
	str := base64.StdEncoding.EncodeToString([]byte(C.GoString(data)))
	return C.CString(string(str))

}

//export waku_utils_free
// Frees a char* since all strings returned by gowaku are allocated in the C heap using malloc.
func waku_utils_free(data *C.char) {
	C.free(unsafe.Pointer(data))
}
