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
		return "ip4or6"
	case IP4:
		return "ip4"
	case IP6:
		return "ip6"
	default:
		return "ip?"
	}
}
