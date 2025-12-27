package models

// HTMLData is a list of HTML fields to be rendered.
// It is exported so that the HTML template engine can render it.
type HTMLData struct {
	Rows          []HTMLRow
	TotalDomains  int
	SuccessCount  int
	ErrorCount    int
	UpdatingCount int
	LastUpdate    string
	PublicIPv4    string
	PublicIPv6    string
}

// HTMLRow contains HTML fields to be rendered
// It is exported so that the HTML template engine can render it.
type HTMLRow struct {
	Domain      string
	Owner       string
	Provider    string
	IPVersion   string
	Status      string
	StatusClass string // CSS class for row background tinting (status-success, status-error, etc.)
	CurrentIP   string
	PreviousIPs string
}
