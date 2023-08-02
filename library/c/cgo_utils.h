#include <stdlib.h>
#include <stdint.h>

typedef void (*WakuCallBack) (const char* msg, size_t len_0);

typedef void (*BytesCallBack) (uint8_t* data, size_t len_0);