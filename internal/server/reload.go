package server

import (
	"fmt"
	"net/http"
	"time"
)

func (h *handlers) reload(w http.ResponseWriter, r *http.Request) {
	// An optional period query parameter allows setting the period
	// between two public IP address checks, for example
	// /reload?period=30s
	if periodString := r.URL.Query().Get("period"); periodString != "" {
		period, err := time.ParseDuration(periodString)
		if err != nil {
			httpError(w, http.StatusBadRequest,
				"invalid period "+periodString+": "+err.Error())
			return
		}
		if period <= 0 {
			httpError(w, http.StatusBadRequest,
				"invalid period "+periodString+": it must be positive")
			return
		}

		err = h.reloader.SetPeriod(period)
		if err != nil {
			httpError(w, http.StatusInternalServerError,
				"failed to set period: "+err.Error())
			return
		}
	}

	result, err := h.reloader.Reload(h.ctx)
	if err != nil {
		httpError(w, http.StatusInternalServerError,
			"failed to reload configuration: "+err.Error())
		return
	}

	message := fmt.Sprintf("Reloaded %d records", result.RecordCount)
	if result.PeriodSet {
		message += "; period is " + result.Period.String()
	}
	if result.CooldownSet {
		message += "; cooldown is " + result.Cooldown.String()
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(message))
}
