package server

import (
	"net/http"
)

func (h *handlers) update(w http.ResponseWriter, r *http.Request) {
	// TODO make RESTful = blocking and not fire and forget
	// By using two channels or abstracting it in a function call
	// which waits for an update cycle to finish and returns an error or nil.
	// Then we can return the result of the update to the user via HTTP.
	h.forceUpdate <- struct{}{}
	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write([]byte(`<b>Update launched</b>`))
}
