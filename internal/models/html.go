package models

// HTMLData is a list of HTML fields to be rendered.
// It is exported so that the HTML template engine can render it.
type HTMLData struct {
	Rows []HTMLRow
}

// HTMLRow contains HTML fields to be rendered
// It is exported so that the HTML template engine can render it.
type HTMLRow struct {
	Domain      HTML
	Host        HTML
	Provider    HTML
	IPVersion   HTML
	Status      HTML
	CurrentIP   HTML
	PreviousIPs HTML
}
