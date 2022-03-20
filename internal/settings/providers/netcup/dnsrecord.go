package netcup

type DNSRecord struct {
	ID           string `json:"id"`
	Hostname     string `json:"hostname"`
	Type         string `json:"type"`
	Priority     string `json:"priority"`
	Destination  string `json:"destination"`
	DeleteRecord bool   `json:"deleterecord"`
	State        string `json:"state"`
}

func NewDNSRecord(hostname, dnstype, destination string) *DNSRecord {
	return &DNSRecord{
		Hostname:    hostname,
		Type:        dnstype,
		Destination: destination,
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
