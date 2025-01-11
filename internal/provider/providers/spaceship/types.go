package spaceship

// APIError represents the Spaceship API error response
type APIError struct {
	Detail string `json:"detail"`
	Data   []struct {
		Field   string `json:"field"`
		Details string `json:"details"`
	} `json:"data"`
}

// Record represents a DNS record
type Record struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Address string `json:"address"`
}
