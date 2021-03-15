package ipversion

type IPVersion uint8

const (
	IP4or6 IPVersion = iota
	IP4
	IP6
)

func (v IPVersion) String() string {
	switch v {
	case IP4or6:
		return "ipv4 or ipv6"
	case IP4:
		return "ipv4"
	case IP6:
		return "ipv6"
	default:
		return "ip?"
	}
}
