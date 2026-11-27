package httpserver

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// WriteJSON writes value as application/json with status.
func WriteJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

// DecodeJSON decodes the request body into destination. The body must hold exactly one JSON value with no fields
// unknown to destination. DecodeJSON returns nil on success, a 413 body_too_large problem when the body exceeds an
// http.MaxBytesReader limit, and a 400 malformed_json problem for any other failure.
func DecodeJSON(r *http.Request, destination any) *Problem {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(destination)
	if err == nil {
		if err = decoder.Decode(&struct{}{}); errors.Is(err, io.EOF) {
			return nil
		}
	}

	if errors.As(err, new(*http.MaxBytesError)) {
		return &Problem{
			Status: http.StatusRequestEntityTooLarge,
			Code:   "body_too_large",
			Detail: "request body is too large",
		}
	}
	return &Problem{
		Status: http.StatusBadRequest,
		Code:   "malformed_json",
		Detail: "request body must be one JSON object with the documented fields",
	}
}
