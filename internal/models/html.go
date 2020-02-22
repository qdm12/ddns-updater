package models

// HTMLData is a list of HTML fields to be rendered.
// It is exported so that the HTML template engine can render it.
type HTMLData struct {
	Rows []HTMLRow
}

// HTMLRow contains HTML fields to be rendered
// It is exported so that the HTML template engine can render it.
type HTMLRow struct {
	Domain      string
	Host        string
	Provider    string
	IPMethod    string
	Status      string
	CurrentIP   string
	PreviousIPs string
}
