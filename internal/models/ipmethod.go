package models

import "fmt"

// IPMethodType is the enum type for all the possible IP methods
type IPMethodType string

// All possible IP methods values
const (
	IPMETHODPROVIDER IPMethodType = "provider"
	IPMETHODGOOGLE                = "google"
	IPMETHODOPENDNS               = "opendns"
)

// ParseIPMethod obtains the IP method from a string
func ParseIPMethod(s string) (IPMethodType, error) {
	switch s {
	case "provider", "google", "opendns":
		return IPMethodType(s), nil
	case "duckduckgo":
		return "", fmt.Errorf("IP method duckduckgo no longer supported")
	default:
		return "", fmt.Errorf("IP method %s not recognized", s)
	}
}
