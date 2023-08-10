#include <stdlib.h>
#include <stddef.h>
#include <cgo_utils.h>

// This is a bridge function to execute C callbacks.
// It's used internally in go-waku. Do not call directly
void _waku_execCB(WakuCallBack op, char* a, size_t b) {
    op(a, b);
}
