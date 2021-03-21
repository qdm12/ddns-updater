package publicip

type FetchType uint8

const (
	DNS FetchType = iota
	HTTP
)
