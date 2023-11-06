package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func writeErrOrResponse(w http.ResponseWriter, err error, value interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	jsonResponse, err := json.Marshal(value)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, err = w.Write(jsonResponse)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func writeResponse(w http.ResponseWriter, value interface{}, code int) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	jsonResponse, err := json.Marshal(value)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// w.Write implicitly writes a 200 status code
	// and only once we can write 2xx-5xx status code
	// so any statusCode apart from 1xx being written to the header, will be ignored.
	w.WriteHeader(code)
	_, _ = w.Write(jsonResponse)
}

func topicFromPath(w http.ResponseWriter, req *http.Request, field string, logger *zap.Logger) string {
	topic := chi.URLParam(req, field)
	if topic == "" {
		errMissing := fmt.Errorf("missing %s", field)
		writeGetMessageErr(w, errMissing, http.StatusBadRequest, logger)
		return ""
	}
	topic, err := url.QueryUnescape(topic)
	if err != nil {
		errInvalid := fmt.Errorf("invalid %s format", field)
		writeGetMessageErr(w, errInvalid, http.StatusBadRequest, logger)
		return ""
	}
	return topic
}

func writeGetMessageErr(w http.ResponseWriter, err error, code int, logger *zap.Logger) {
	// write status before the body
	w.WriteHeader(code)
	logger.Error("get message", zap.Error(err))
	if _, err := w.Write([]byte(err.Error())); err != nil {
		logger.Error("writing response", zap.Error(err))
	}
}
