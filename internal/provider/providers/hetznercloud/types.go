package hetznercloud

// recordValue represents a single DNS record value.
type recordValue struct {
	Value string `json:"value"`
}

// recordsRequest represents the request body for creating/updating DNS records.
type recordsRequest struct {
	TTL     uint32        `json:"ttl,omitempty"`
	Records []recordValue `json:"records"`
}

// actionResponse represents the response from Hetzner Cloud API actions.
type actionResponse struct {
	Action struct {
		ID     uint64 `json:"id"`
		Status string `json:"status"`
	} `json:"action"`
}
