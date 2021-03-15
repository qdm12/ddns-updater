package server

import (
	"encoding/json"
	"net/http"
)

func (h *handlers) getRecords(w http.ResponseWriter, r *http.Request) {
	records := h.db.SelectAll()
	encoder := json.NewEncoder(w)
	if err := encoder.Encode(records); err != nil {
		h.logger.Error(err)
		httpError(w, http.StatusInternalServerError, "")
	}
}
