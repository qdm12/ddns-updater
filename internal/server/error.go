package server

import (
	"encoding/json"
	"net/http"
)

type errJSONWrapper struct {
	Error string `json:"error"`
}

func httpError(w http.ResponseWriter, status int, errString string) {
	w.WriteHeader(status)
	if errString == "" {
		errString = http.StatusText(status)
	}
	body := errJSONWrapper{Error: errString}
	_ = json.NewEncoder(w).Encode(body)
}
