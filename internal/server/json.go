package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/qdm12/ddns-updater/internal/models"
)

func (h *handlers) json(w http.ResponseWriter, _ *http.Request) {
	jsonData := models.JSONData{
		Records: []models.JSONRecord{},
		Time:    h.timeNow(),
	}

	now := h.timeNow()
	var lastSuccessTime time.Time
	var lastSuccessIP string

	for _, record := range h.db.SelectAll() {
		jsonRecord := record.JSON(now)
		jsonData.Records = append(jsonData.Records, jsonRecord)

		// Track the most recent successful update across all records
		successTime := record.History.GetSuccessTime()
		if !successTime.IsZero() && successTime.After(lastSuccessTime) {
			lastSuccessTime = successTime
			currentIP := record.History.GetCurrentIP()
			if currentIP.IsValid() {
				lastSuccessIP = currentIP.String()
			}
		}
	}

	jsonData.LastSuccessTime = lastSuccessTime
	jsonData.LastSuccessIP = lastSuccessIP

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")

	err := json.NewEncoder(w).Encode(jsonData)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "failed encoding JSON: "+err.Error())
		return
	}
}

