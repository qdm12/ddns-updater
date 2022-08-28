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
	b, err := json.Marshal(r)
	if err != nil {
		panic(err)
	}

	return string(b)
}
