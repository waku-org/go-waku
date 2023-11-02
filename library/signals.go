package library

/*
#include <stddef.h>
#include <stdbool.h>
#include <stdlib.h>
extern bool ServiceSignalEvent(const char *jsonEvent);
extern void SetEventCallback(void *cb);
*/
import "C"

import (
	"encoding/json"
	"fmt"
	"unsafe"
)

// MobileSignalHandler is a simple callback function that gets called when any signal is received
type MobileSignalHandler func([]byte)

// storing the current mobile signal handler here
var mobileSignalHandler MobileSignalHandler

// signalEnvelope is a general signal sent upward from node to app
type signalEnvelope struct {
	Type  string      `json:"type"`
	Event interface{} `json:"event"`
}

// NewEnvelope creates new envlope of given type and event payload.
func newEnvelope(signalType string, event interface{}) *signalEnvelope {
	return &signalEnvelope{
		Type:  signalType,
		Event: event,
	}
}

// send sends application signal (in JSON) upwards to application (via default notification handler)
func send(signalType string, event interface{}) {

	signal := newEnvelope(signalType, event)
	data, err := json.Marshal(&signal)
	if err != nil {
		fmt.Println("marshal signal error", err)
		return
	}
	// If a Go implementation of signal handler is set, let's use it.
	if mobileSignalHandler != nil {
		mobileSignalHandler(data)
	} else {
		// ...and fallback to C implementation otherwise.
		dataStr := string(data)
		str := C.CString(dataStr)
		C.ServiceSignalEvent(str)
		C.free(unsafe.Pointer(str))
	}
}

// SetEventCallback is to set a callback in order to receive application
// signals which are used to react to asynchronous events in waku.
func SetEventCallback(cb unsafe.Pointer) {
	C.SetEventCallback(cb)
}

// SetMobileSignalHandler sets the callback to be executed when a signal
// is received in a mobile device
func SetMobileSignalHandler(m MobileSignalHandler) {
	mobileSignalHandler = m
}
