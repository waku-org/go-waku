// ======================================================================================
// cgo compilation (for desktop platforms and local tests)
// ======================================================================================

#include <stdio.h>
#include <stddef.h>
#include <stdbool.h>
#include "_cgo_export.h"

typedef void (*callback)(const char *jsonEvent, size_t len_0);
callback gCallback = 0;

bool ServiceSignalEvent(const char *jsonEvent, size_t len_0) {
	if (gCallback) {
		gCallback(jsonEvent, len_0);
	}

	return true;
}

void SetEventCallback(void *cb) {
	gCallback = (callback)cb;
}
