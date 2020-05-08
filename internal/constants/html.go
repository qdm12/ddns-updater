package constants

import "github.com/qdm12/ddns-updater/internal/models"

const (
	HTMLFail     models.HTML = `<font color="red"><b>Failure</b></font>`
	HTMLSuccess  models.HTML = `<font color="green"><b>Success</b></font>`
	HTMLUpdate   models.HTML = `<font color="#00CC66"><b>Up to date</b></font>`
	HTMLUpdating models.HTML = `<font color="orange"><b>Updating</b></font>`
)

const (
	// TODO have a struct model containing URL, name for each provider
	HTMLNamecheap  models.HTML = "<a href=\"https://namecheap.com\">Namecheap</a>"
	HTMLGodaddy    models.HTML = "<a href=\"https://godaddy.com\">GoDaddy</a>"
	HTMLDuckDNS    models.HTML = "<a href=\"https://duckdns.org\">DuckDNS</a>"
	HTMLDreamhost  models.HTML = "<a href=\"https://www.dreamhost.com/\">Dreamhost</a>"
	HTMLCloudflare models.HTML = "<a href=\"https://www.cloudflare.com\">Cloudflare</a>"
	HTMLNoIP       models.HTML = "<a href=\"https://www.noip.com/\">NoIP</a>"
	HTMLDNSPod     models.HTML = "<a href=\"https://www.dnspod.cn/\">DNSPod</a>"
	HTMLInfomaniak models.HTML = "<a href=\"https://www.infomaniak.com/\">Infomaniak</a>"
	HTMLDdnssde    models.HTML = "<a href=\"https://ddnss.de/\">DDNSS.de</a>"
	HTMLDyn        models.HTML = "<a href=\"https://dyn.com/\">Dyn DNS</a>"
)

const (
	HTMLGoogle   models.HTML = "<a href=\"https://google.com/search?q=ip\">Google</a>"
	HTMLOpenDNS  models.HTML = "<a href=\"https://diagnostic.opendns.com/myip\">OpenDNS</a>"
	HTMLIfconfig models.HTML = "<a href=\"https://ifconfig.io\">ifconfig.io</a>"
	HTMLIpinfo   models.HTML = "<a href=\"https://ipinfo.io\">ipinfo.io</a>"
	HTMLIpify    models.HTML = "<a href=\"https://api.ipify.org\">api.ipify.org</a>"
	HTMLIpify6   models.HTML = "<a href=\"https://api6.ipify.org\">api6.ipify.org</a>"
	HTMLDdnss    models.HTML = "<a href=\"https://ddnss.de/meineip.php\">ddnss.de</a>"
	HTMLDdnss4   models.HTML = "<a href=\"https://ip4.ddnss.de/meineip.php\">ip4.ddnss.de</a>"
	HTMLDdnss6   models.HTML = "<a href=\"https://ip6.ddnss.de/meineip.php\">ip6.ddns.de</a>"
	HTMLCycle    models.HTML = "Cycling"
)
