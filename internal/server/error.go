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

type errorsJSONWrapper struct {
	Errors []string `json:"errors"`
}

func httpErrors(w http.ResponseWriter, status int, errors []error) {
	w.WriteHeader(status)

	errs := make([]string, len(errors))
	for i := range errors {
		errs[i] = errors[i].Error()
	}

	body := errorsJSONWrapper{Errors: errs}
	_ = json.NewEncoder(w).Encode(body)
}
