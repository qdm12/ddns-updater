package info

import "net/netip"

type Result struct {
	IP      netip.Addr
	Country *string
	Region  *string
	City    *string
	Source  string
}

func stringPtr(s string) *string { return &s }
