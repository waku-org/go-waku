package main

/*
#include <stdlib.h>
#include <cgo_utils.h>
extern void _waku_execCB(WakuCallBack op, char* a, size_t b);
*/
import "C"
import "unsafe"

func execOkCB(onOkCb C.WakuCallBack, value string) C.int {
	if onOkCb == nil {
		return retMissingCallback
	}

	val := C.CString(value)
	len := C.size_t(len(value))
	C._waku_execCB(onOkCb, val, len)

	C.free(unsafe.Pointer(val))

	return retOk
}

func execErrCB(onErrCb C.WakuCallBack, err error) C.int {
	if onErrCb == nil {
		return retMissingCallback
	}

	if err != nil {
		errMsg := err.Error()
		execOkCB(onErrCb, errMsg) // reusing ok cb
		return retErr
	}

	return retOk
}
