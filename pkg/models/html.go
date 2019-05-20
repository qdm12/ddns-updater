package models

// HTMLData is a list of HTML fields to be rendered.
// It is exported so that the HTML template engine can render it.
type HTMLData struct {
	Rows []HTMLRow
}

// HTMLRow contains HTML fields to be rendered
// It is exported so that the HTML template engine can render it.
type HTMLRow struct {
	Domain   string
	Host     string
	Provider string
	IPMethod string
	Status   string
	IP       string   // current set ip
	IPs      []string // previous ips
}

// ToHTML converts all the update record configs to HTML data ready to be templated
func ToHTML(recordsConfigs []RecordConfigType) (htmlData HTMLData) {
	for i := range recordsConfigs {
		htmlData.Rows = append(htmlData.Rows, recordsConfigs[i].toHTML())
	}
	return htmlData
}
