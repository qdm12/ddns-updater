package hetznernetworking

// recordValue represents a single DNS record value
type recordValue struct {
	Value string `json:"value"`
}

// recordsRequest represents the request body for creating/updating DNS records
type recordsRequest struct {
	TTL     uint32        `json:"ttl,omitempty"`
	Records []recordValue `json:"records"`
}

// actionResponse represents the response from Hetzner Networking API actions
type actionResponse struct {
	Action struct {
		ID     int    `json:"id"`
		Status string `json:"status"`
	} `json:"action"`
}

// rrSetResponse represents the response from Hetzner Networking API RRSet GET requests
type rrSetResponse struct {
	RRSet struct {
		ID      string `json:"id"`
		Records []struct {
			Value string `json:"value"`
		} `json:"records"`
	} `json:"rrset"`
}
