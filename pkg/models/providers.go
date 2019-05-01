package models

import (
	"fmt"
)

// ProviderType is the enum type for the possible providers
type ProviderType uint8

// All possible provider values
const (
	PROVIDERGODADDY ProviderType = iota
	PROVIDERNAMECHEAP
	PROVIDERDUCKDNS
	PROVIDERDREAMHOST
)

func (provider ProviderType) String() string {
	switch provider {
	case PROVIDERGODADDY:
		return "godaddy"
	case PROVIDERNAMECHEAP:
		return "namecheap"
	case PROVIDERDUCKDNS:
		return "duckdns"
	case PROVIDERDREAMHOST:
		return "dreamhost"
	default:
		return "unknown"
	}
}

// ParseProvider obtains the provider from a string
func ParseProvider(s string) (ProviderType, error) {
	switch s {
	case "godaddy":
		return PROVIDERGODADDY, nil
	case "namecheap":
		return PROVIDERNAMECHEAP, nil
	case "duckdns":
		return PROVIDERDUCKDNS, nil
	case "dreamhost":
		return PROVIDERDREAMHOST, nil
	}
	return 0, fmt.Errorf("Provider %s not recognized", s)
}
