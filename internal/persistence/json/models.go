package json

import (
	"encoding/json"

	"github.com/qdm12/ddns-updater/internal/models"
)

type dataModel struct {
	Records []record `json:"records"`
}

type record struct {
	Domain string                `json:"domain"`
	Host   string                `json:"host"`
	Events []models.HistoryEvent `json:"ips"`
}

func (r record) String() string {
	b, _ := json.Marshal(r)
	return string(b)
}
