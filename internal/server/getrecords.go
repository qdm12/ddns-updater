package server

import (
	"encoding/json"
	"net/http"
)

func (h *handlers) getRecords(w http.ResponseWriter, r *http.Request) {
	accept := r.Header.Get("Accept")
	var contentType string
	switch accept {
	case "", "application/json":
		contentType = "application/json"
	default:
		httpError(w, http.StatusBadRequest, `content type "`+accept+`" is not supported`)
	}
	w.Header().Set("Content-Type", contentType)

	records := h.db.SelectAll()
	encoder := json.NewEncoder(w)
	// TODO check Accept header and return Content-Type header
	if err := encoder.Encode(records); err != nil {
		h.logger.Error(err)
		httpError(w, http.StatusInternalServerError, "")
	}
}
