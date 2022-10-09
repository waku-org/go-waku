package main

/*
#include <stdlib.h>
#include <stddef.h>
*/
import "C"
import (
	"encoding/base64"
	"unsafe"

	mobile "github.com/status-im/go-waku/mobile"
)

// Decode a base64 string (useful for reading the payload from waku messages)
//
//export waku_utils_base64_decode
func waku_utils_base64_decode(data *C.char) *C.char {
	b, err := base64.StdEncoding.DecodeString(C.GoString(data))
	if err != nil {
		return C.CString(mobile.MakeJSONResponse(err))
	}

	return prepareJSONResponse(string(b), nil)
}

// Encode data to base64 (useful for creating the payload of a waku message in the
// format understood by waku_relay_publish)
//
//export waku_utils_base64_encode
func waku_utils_base64_encode(data *C.char) *C.char {
	str := base64.StdEncoding.EncodeToString([]byte(C.GoString(data)))
	return C.CString(string(str))

}

// Frees a char* since all strings returned by gowaku are allocated in the C heap using malloc.
//
//export waku_utils_free
func waku_utils_free(data *C.char) {
	C.free(unsafe.Pointer(data))
}
