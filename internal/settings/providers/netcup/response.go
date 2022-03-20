package netcup

import (
	"encoding/json"
)

type LoginResponse struct {
	Session string `json:"session"`
}

type NetcupResponse struct {
	ServerRequestID string          `json:"serverrequestid"`
	ClientRequestID string          `json:"clientrequestid"`
	Action          string          `json:"action"`
	Status          string          `json:"status"`
	StatusCode      int             `json:"statuscode"`
	ShortMessage    string          `json:"shortmessage"`
	LongMessage     string          `json:"longmessage"`
	ResponseData    json.RawMessage `json:"responsedata"`
}

func (r *NetcupResponse) isSuccess() bool {
	return r.Status == "success"
}

func (r *NetcupResponse) isError() bool {
	return r.Status == "error"
}
