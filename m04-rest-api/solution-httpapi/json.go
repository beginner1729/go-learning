package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

const maxBodyBytes = 1 << 20 // 1 MiB

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status) // must precede Write
	_ = json.NewEncoder(w).Encode(v)
}

// decodeJSON safely decodes a single JSON object into T with strict rules.
func decodeJSON[T any](w http.ResponseWriter, r *http.Request) (T, error) {
	var v T
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&v); err != nil {
		return v, fmt.Errorf("decode body: %w", err)
	}
	if dec.More() {
		return v, errors.New("body must contain a single JSON object")
	}
	return v, nil
}
