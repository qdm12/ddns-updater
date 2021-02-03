package server

import (
	"net/http"
)

func (h *handlers) update(w http.ResponseWriter, r *http.Request) {
	start := h.timeNow()
	errors := h.runner.ForceUpdate()
	duration := h.timeNow().Sub(start)
	if len(errors) > 0 {
		httpErrors(w, http.StatusInternalServerError, errors)
		return
	}
	w.WriteHeader(http.StatusAccepted)
	message := "All records updated successfully in " + duration.String()
	_, _ = w.Write([]byte(message))
}
