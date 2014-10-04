package handler

import (
	"encoding/json"
	"net/http"
)

func renderJSON(w http.ResponseWriter, v interface{}, status int) error {
	w.WriteHeader(status)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return json.NewEncoder(w).Encode(&v)
}
