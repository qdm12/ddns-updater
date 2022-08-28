package health

import "github.com/qdm12/ddns-updater/internal/records"

type AllSelecter interface {
	SelectAll() (records []records.Record)
}
