package vultr

type Record struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	IP       string `json:"data"`
	Priority int32  `json:"priority"`
	TTL      uint32 `json:"ttl"`
}
