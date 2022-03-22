package main

/*
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
//nolint
type SignalHandler interface {
	HandleSignal(string)
}

// SignalHandler is a simple callback function that gets called when any signal is received
type MobileSignalHandler func([]byte)

// storing the current mobile signal handler here
var mobileSignalHandler MobileSignalHandler

// SignalEnvelope is a general signal sent upward from node to app
type SignalEnvelope struct {
	NodeID int         `json:"nodeId"`
	Type   string      `json:"type"`
	Event  interface{} `json:"event"`
}

// NewEnvelope creates new envlope of given type and event payload.
func NewEnvelope(nodeId int, typ string, event interface{}) *SignalEnvelope {
	return &SignalEnvelope{
		NodeID: nodeId,
		Type:   typ,
		Event:  event,
	}
}

// send sends application signal (in JSON) upwards to application (via default notification handler)
func send(node int, typ string, event interface{}) {
	signal := NewEnvelope(node, typ, event)
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
//nolint
func SetMobileSignalHandler(handler SignalHandler) {
	mobileSignalHandler = func(data []byte) {
		if len(data) > 0 {
			handler.HandleSignal(string(data))
		}
	}
}

func setEventCallback(cb unsafe.Pointer) {
	C.SetEventCallback(cb)
}
