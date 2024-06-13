package server

import (
	"net/http"
)

func (h *handlers) update(w http.ResponseWriter, _ *http.Request) {
	start := h.timeNow()
	errors := h.runner.ForceUpdate(h.ctx) //nolint:contextcheck
	duration := h.timeNow().Sub(start)
	if len(errors) > 0 {
		httpErrors(w, http.StatusInternalServerError, errors)
		return
	}
	w.WriteHeader(http.StatusAccepted)
	message := "All records updated successfully in " + duration.String()
	_, _ = w.Write([]byte(message))
}
