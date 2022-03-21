package main

import (
	"bytes"
	"fmt"
	"strings"
)

// APIResponse generic response from API.
type APIResponse struct {
	Error *string `json:"error"`
}

// APIDetailedResponse represents a generic response
// with possible errors.
//nolint
type APIDetailedResponse struct {
	Status      bool            `json:"status"`
	Message     string          `json:"message,omitempty"`
	FieldErrors []APIFieldError `json:"field_errors,omitempty"`
}

// Error string representation of APIDetailedResponse.
//nolint
func (r APIDetailedResponse) Error() string {
	buf := bytes.NewBufferString("")

	for _, err := range r.FieldErrors {
		buf.WriteString(err.Error() + "\n") // nolint: gas
	}

	return strings.TrimSpace(buf.String())
}

// APIFieldError represents a set of errors
// related to a parameter.
//nolint
type APIFieldError struct {
	Parameter string     `json:"parameter,omitempty"`
	Errors    []APIError `json:"errors"`
}

// Error string representation of APIFieldError.
func (e APIFieldError) Error() string {
	if len(e.Errors) == 0 {
		return ""
	}

	buf := bytes.NewBufferString(fmt.Sprintf("Parameter: %s\n", e.Parameter))

	for _, err := range e.Errors {
		buf.WriteString(err.Error() + "\n") // nolint: gas
	}

	return strings.TrimSpace(buf.String())
}

// APIError represents a single error.
//nolint
type APIError struct {
	Message string `json:"message"`
}

// Error string representation of APIError.
func (e APIError) Error() string {
	return fmt.Sprintf("message=%s", e.Message)
}

// SignalHandler defines a minimal interface
// a signal handler needs to implement.
//nolint
type SignalHandler interface {
	HandleSignal(string)
}
