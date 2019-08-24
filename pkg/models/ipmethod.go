package models

import "fmt"

// IPMethodType is the enum type for all the possible IP methods
type IPMethodType uint8

// All possible IP methods values
const (
	IPMETHODPROVIDER IPMethodType = iota
	IPMETHODDUCKDUCKGO
	// IPMETHODOPENDNS
)

func (ipMethod IPMethodType) String() string {
	switch ipMethod {
	case IPMETHODPROVIDER:
		return "provider"
	case IPMETHODDUCKDUCKGO:
		return "duckduckgo"
	// case IPMETHODOPENDNS:
	// 	return "opendns"
	default:
		return "unknown"
	}
}

// ParseIPMethod obtains the IP method from a string
func ParseIPMethod(s string) (IPMethodType, error) {
	switch s {
	case "provider":
		return IPMETHODPROVIDER, nil
	case "duckduckgo":
		return IPMETHODDUCKDUCKGO, nil
	case "opendns":
		return 0, fmt.Errorf("IP method %s no longer supported", s)
	}
	return 0, fmt.Errorf("IP method %s not recognized", s)
}
