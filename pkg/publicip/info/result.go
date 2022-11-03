package info

import "net"

type Result struct {
	IP      net.IP
	Country *string
	Region  *string
	City    *string
	Source  string
}

func stringPtr(s string) *string { return &s }
