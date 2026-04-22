package spaceship

// apiError represents the Spaceship API error response.
type apiError struct {
	Detail string `json:"detail"`
	Data   []struct {
		Field   string `json:"field"`
		Details string `json:"details"`
	} `json:"data"`
}

// apiRecord represents a DNS record.
type apiRecord struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Address string `json:"address"`
	TTL     uint32 `json:"ttl,omitempty"`
}
