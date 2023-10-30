// ======================================================================================
// cgo compilation (for desktop platforms and local tests)
// ======================================================================================

#include <stdio.h>
#include <stddef.h>
#include <stdbool.h>
#include "_cgo_export.h"

typedef void (*callback)(int retCode, const char *jsonEvent, void* userData);
callback gCallback = 0;

bool ServiceSignalEvent(const char *jsonEvent) {
	if (gCallback) {
		gCallback(0, jsonEvent, NULL);
	}

	return true;
}

void SetEventCallback(void *cb) {
	gCallback = (callback)cb;
}
