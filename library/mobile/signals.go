package gowaku

import "github.com/waku-org/go-waku/library"

// SignalHandler defines a minimal interface
// a signal handler needs to implement.
// nolint
type SignalHandler interface {
	HandleSignal(string)
}

// SetMobileSignalHandler setup geth callback to notify about new signal
// used for gomobile builds
// nolint
func SetMobileSignalHandler(handler SignalHandler) {
	library.SetMobileSignalHandler(func(data []byte) {
		if len(data) > 0 {
			handler.HandleSignal(string(data))
		}
	})
}
