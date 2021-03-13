package publicip

type fetchType uint8

const (
	dnsFetch fetchType = iota
	httpFetch
)
