package constants

import "github.com/qdm12/ddns-updater/internal/models"

const (
	HTML_FAIL     models.HTML = `<font color="red"><b>Failure</b></font>`
	HTML_SUCCESS  models.HTML = `<font color="green"><b>Success</b></font>`
	HTML_UPTODATE models.HTML = `<font color="#00CC66"><b>Up to date</b></font>`
	HTML_UPDATING models.HTML = `<font color="orange"><b>Updating</b></font>`
)

const (
	// TODO have a struct model containing URL, name for each provider
	HTML_NAMECHEAP  models.HTML = "<a href=\"https://namecheap.com\">Namecheap</a>"
	HTML_GODADDY    models.HTML = "<a href=\"https://godaddy.com\">GoDaddy</a>"
	HTML_DUCKDNS    models.HTML = "<a href=\"https://duckdns.org\">DuckDNS</a>"
	HTML_DREAMHOST  models.HTML = "<a href=\"https://www.dreamhost.com/\">Dreamhost</a>"
	HTML_CLOUDFLARE models.HTML = "<a href=\"https://www.cloudflare.com\">Cloudflare</a>"
	HTML_NOIP       models.HTML = "<a href=\"https://www.noip.com/\">NoIP</a>"
	HTML_DNSPOD     models.HTML = "<a href=\"https://www.dnspod.cn/\">DNSPod</a>"
	HTML_INFOMANIAK models.HTML = "<a href=\"https://www.infomaniak.com/\">Infomaniak</a>"
)

const (
	HTML_GOOGLE   models.HTML = "<a href=\"https://google.com/search?q=ip\">Google</a>"
	HTML_OPENDNS  models.HTML = "<a href=\"https://diagnostic.opendns.com/myip\">OpenDNS</a>"
	HTML_IFCONFIG models.HTML = "<a href=\"https://ifconfig.io\">ifconfig.io</a>"
	HTML_IPINFO   models.HTML = "<a href=\"https://ipinfo.io\">ipinfo.io</a>"
	HTML_IPIFY    models.HTML = "<a href=\"https://api.ipify.org\">api.ipify.org</a>"
	HTML_IPIFY6   models.HTML = "<a href=\"https://api6.ipify.org\">api6.ipify.org</a>"
	HTML_DDNSS    models.HTML = "<a href=\"https://ddnss.de/meineip.php\">ddns.de</a>"
	HTML_DDNSS4   models.HTML = "<a href=\"https://ip4.ddnss.de/meineip.php\">ip4.ddns.de</a>"
	HTML_DDNSS6   models.HTML = "<a href=\"https://ip6.ddnss.de/meineip.php\">ip6.ddns.de</a>"
	HTML_CYCLE    models.HTML = "Cycling"
)
