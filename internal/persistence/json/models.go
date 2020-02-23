package json

import (
	"encoding/json"
	"net"
	"time"
)

type dataModel struct {
	Records []record `json:"records"`
}

type record struct {
	Domain string   `json:"domain"`
	Host   string   `json:"host"`
	IPs    []ipData `json:"ips"`
}

func (r record) String() string {
	b, _ := json.Marshal(r)
	return string(b)
}

type ipData struct {
	IP   net.IP    `json:"ip"`
	Time time.Time `json:"time"`
}
