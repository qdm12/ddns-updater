package info

type Result struct {
	Country *string
	Region  *string
	City    *string
}

func stringPtr(s string) *string { return &s }
