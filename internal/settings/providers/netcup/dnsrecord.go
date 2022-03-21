package netcup

type DNSRecord struct {
	ID           string `json:"id"`
	DeleteRecord bool   `json:"deleterecord"`
	Destination  string `json:"destination"`
	Hostname     string `json:"hostname"`
	Priority     string `json:"priority"`
	State        string `json:"state"`
	Type         string `json:"type"`
}

func NewDNSRecord(hostname, dnstype, destination string) *DNSRecord {
	return &DNSRecord{
		Destination: destination,
		Hostname:    hostname,
		Type:        dnstype,
	}
}

type DNSRecordSet struct {
	DNSRecords []DNSRecord `json:"dnsrecords"`
}

func NewDNSRecordSet(records []DNSRecord) *DNSRecordSet {
	return &DNSRecordSet{
		DNSRecords: records,
	}
}

func (r *DNSRecordSet) GetRecord(name, dnstype string) *DNSRecord {
	for _, record := range r.DNSRecords {
		if record.Hostname == name && record.Type == dnstype {
			return &record
		}
	}
	return nil
}

func (r *DNSRecordSet) GetRecordOccurences(hostname, dnstype string) int {
	result := 0
	for _, record := range r.DNSRecords {
		if record.Hostname == hostname && record.Type == dnstype {
			result++
		}
	}
	return result
}
