package health

//go:generate mockgen -destination=mocks_test.go -package=$GOPACKAGE . AllSelecter,LookupIPer
//go:generate mockgen -destination=mocks_provider_test.go -package=$GOPACKAGE github.com/qdm12/ddns-updater/internal/provider Provider
