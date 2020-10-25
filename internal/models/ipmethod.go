package models

// IPMethod is a method to obtain your public IP address.
type IPMethod struct {
	Name string
	URL  string
	IPv4 bool
	IPv6 bool
}
