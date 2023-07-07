package gowaku

/*
#include <stddef.h>
#include <stdbool.h>
#include <stdlib.h>
extern bool StatusServiceSignalEvent(const char *jsonEvent);
extern void SetEventCallback(void *cb);
*/
import "C"

import (
	"encoding/json"
	"fmt"
	"unsafe"
)

// SignalHandler defines a minimal interface
// a signal handler needs to implement.
// nolint
type SignalHandler interface {
	HandleSignal(string)
}

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
		str := C.CString(string(data))
		C.StatusServiceSignalEvent(str)
		C.free(unsafe.Pointer(str))
	}
}

// SetMobileSignalHandler setup geth callback to notify about new signal
// used for gomobile builds
// nolint
func SetMobileSignalHandler(handler SignalHandler) {
	mobileSignalHandler = func(data []byte) {
		if len(data) > 0 {
			handler.HandleSignal(string(data))
		}
	}
}

// SetEventCallback is to set a callback in order to receive application
// signals which are used to react to asynchronous events in waku.
func SetEventCallback(cb unsafe.Pointer) {
	C.SetEventCallback(cb)
}
