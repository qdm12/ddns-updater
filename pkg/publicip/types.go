package publicip

type fetchType uint8

const (
	dnsFetch fetchType = iota
	httpFetch
)

func listFetchTypes() []fetchType {
	return []fetchType{
		dnsFetch,
		httpFetch,
	}
}
