package utils

func BuildDomainName(host, domain string) string {
	switch host {
	case "@":
		return domain
	case "*":
		return "any." + domain
	default:
		return host + "." + domain
	}
}
