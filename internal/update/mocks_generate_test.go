package update

//go:generate mockgen -destination=mocks_test.go -package=$GOPACKAGE . Database,HealthchecksIOClient,Logger,LookupIPer,PublicIPFetcher,UpdaterInterface
//go:generate mockgen -destination=mocks_provider_test.go -package=$GOPACKAGE github.com/qdm12/ddns-updater/internal/provider Provider
