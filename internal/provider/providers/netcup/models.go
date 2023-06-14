package netcup

type dnsRecord struct {
	ID          string `json:"id"`
	Destination string `json:"destination"`
	Hostname    string `json:"hostname"`
	Priority    string `json:"priority"`
	State       string `json:"state"`
	Type        string `json:"type"`
}

type dnsRecordSet struct {
	DNSRecords []dnsRecord `json:"dnsrecords"`
}
