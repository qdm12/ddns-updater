package netcup

import (
	"encoding/json"
)

type LoginResponse struct {
	Session string `json:"apisessionid"`
}

type NetcupResponse struct {
	Action          string          `json:"action"`
	ClientRequestID string          `json:"clientrequestid"`
	LongMessage     string          `json:"longmessage"`
	ResponseData    json.RawMessage `json:"responsedata"`
	ServerRequestID string          `json:"serverrequestid"`
	ShortMessage    string          `json:"shortmessage"`
	Status          string          `json:"status"`
	StatusCode      int             `json:"statuscode"`
}

func (r *NetcupResponse) isError() bool {
	return r.Status == "error"
}
