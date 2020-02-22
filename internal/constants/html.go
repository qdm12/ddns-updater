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
)

const (
	HTML_GOOGLE  models.HTML = "<a href=\"https://google.com/search?q=ip\">Google</a>"
	HTML_OPENDNS models.HTML = "<a href=\"https://diagnostic.opendns.com/myip\">OpenDNS</a>"
)
