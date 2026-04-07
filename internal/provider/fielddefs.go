package provider

// FieldDefinition describes a single form field for a provider.
type FieldDefinition struct {
	Name        string   `json:"name"`
	Label       string   `json:"label"`
	Type        string   `json:"type"` // "text", "password", "number", "boolean", "select"
	Required    bool     `json:"required"`
	Placeholder string   `json:"placeholder,omitempty"`
	Help        string   `json:"help,omitempty"`
	Options     []string `json:"options,omitempty"`
}

// AuthGroup represents a set of fields for one authentication method.
type AuthGroup struct {
	Name   string            `json:"name"`
	Fields []FieldDefinition `json:"fields"`
}

// ProviderDefinition describes a provider's form fields for the WebUI.
type ProviderDefinition struct {
	Name       string            `json:"name"`
	URL        string            `json:"url"`
	Fields     []FieldDefinition `json:"fields"`
	AuthGroups []AuthGroup       `json:"auth_groups,omitempty"`
}

// ProviderDefinitions maps provider IDs to their form field definitions.
var ProviderDefinitions = map[string]ProviderDefinition{
	"aliyun": {
		Name: "Aliyun",
		URL:  "https://www.aliyun.com",
		Fields: []FieldDefinition{
			{Name: "access_key_id", Label: "Access Key ID", Type: "password", Required: true, Placeholder: "Your Aliyun access key ID"},
			{Name: "access_secret", Label: "Access Secret", Type: "password", Required: true, Placeholder: "Your Aliyun access secret"},
			{Name: "region", Label: "Region", Type: "text", Required: false, Placeholder: "e.g. cn-hangzhou"},
		},
	},
	"allinkl": {
		Name: "All-Inkl",
		URL:  "https://all-inkl.com",
		Fields: []FieldDefinition{
			{Name: "username", Label: "Username", Type: "text", Required: true, Placeholder: "dynXXXXXXX"},
			{Name: "password", Label: "Password", Type: "password", Required: true},
		},
	},
	"changeip": {
		Name: "ChangeIP",
		URL:  "https://changeip.com",
		Fields: []FieldDefinition{
			{Name: "username", Label: "Username", Type: "text", Required: true},
			{Name: "password", Label: "Password", Type: "password", Required: true},
		},
	},
	"cloudflare": {
		Name: "Cloudflare",
		URL:  "https://www.cloudflare.com",
		Fields: []FieldDefinition{
			{Name: "zone_identifier", Label: "Zone Identifier", Type: "text", Required: true, Placeholder: "e.g. abc123def456", Help: "Found in your Cloudflare dashboard under Overview"},
			{Name: "ttl", Label: "TTL", Type: "number", Required: true, Placeholder: "1", Help: "Set to 1 for automatic"},
			{Name: "proxied", Label: "Proxied", Type: "boolean", Required: false, Help: "Enable Cloudflare proxy"},
		},
		AuthGroups: []AuthGroup{
			{
				Name: "API Token (recommended)",
				Fields: []FieldDefinition{
					{Name: "token", Label: "API Token", Type: "password", Required: true, Placeholder: "Your Cloudflare API token"},
				},
			},
			{
				Name: "Global API Key",
				Fields: []FieldDefinition{
					{Name: "email", Label: "Email", Type: "text", Required: true},
					{Name: "key", Label: "Global API Key", Type: "password", Required: true},
				},
			},
			{
				Name: "User Service Key",
				Fields: []FieldDefinition{
					{Name: "user_service_key", Label: "User Service Key", Type: "password", Required: true},
				},
			},
		},
	},
	"custom": {
		Name: "Custom",
		URL:  "",
		Fields: []FieldDefinition{
			{Name: "url", Label: "URL", Type: "text", Required: true, Placeholder: "https://example.com/update?ip=%s", Help: "URL template for updating DNS"},
			{Name: "ipv4key", Label: "IPv4 Key", Type: "text", Required: false, Placeholder: "Query parameter name for IPv4"},
			{Name: "ipv6key", Label: "IPv6 Key", Type: "text", Required: false, Placeholder: "Query parameter name for IPv6"},
			{Name: "success_regex", Label: "Success Regex", Type: "text", Required: true, Placeholder: "e.g. ^(ok|good)", Help: "Regex pattern to match a successful response body"},
		},
	},
	"dd24": {
		Name: "DD24",
		URL:  "https://www.domaindiscount24.com",
		Fields: []FieldDefinition{
			{Name: "password", Label: "Password", Type: "password", Required: true},
		},
	},
	"ddnss": {
		Name: "DDNSS.de",
		URL:  "https://ddnss.de",
		Fields: []FieldDefinition{
			{Name: "username", Label: "Username", Type: "text", Required: true},
			{Name: "password", Label: "Password", Type: "password", Required: true},
			{Name: "dual_stack", Label: "Dual Stack", Type: "boolean", Required: false, Help: "Update both IPv4 and IPv6 simultaneously"},
		},
	},
	"desec": {
		Name: "deSEC",
		URL:  "https://desec.io",
		Fields: []FieldDefinition{
			{Name: "token", Label: "Token", Type: "password", Required: true},
		},
	},
	"digitalocean": {
		Name: "DigitalOcean",
		URL:  "https://www.digitalocean.com",
		Fields: []FieldDefinition{
			{Name: "token", Label: "API Token", Type: "password", Required: true},
		},
	},
	"dnsomatic": {
		Name: "DNS-O-Matic",
		URL:  "https://www.dnsomatic.com",
		Fields: []FieldDefinition{
			{Name: "username", Label: "Username", Type: "text", Required: true},
			{Name: "password", Label: "Password", Type: "password", Required: true},
		},
	},
	"dnspod": {
		Name: "DNSPod",
		URL:  "https://www.dnspod.cn",
		Fields: []FieldDefinition{
			{Name: "token", Label: "Token", Type: "password", Required: true},
		},
	},
	"domeneshop": {
		Name: "Domeneshop",
		URL:  "https://www.domeneshop.no",
		Fields: []FieldDefinition{
			{Name: "token", Label: "Token", Type: "password", Required: true},
			{Name: "secret", Label: "Secret", Type: "password", Required: true},
		},
	},
	"dondominio": {
		Name: "Don Dominio",
		URL:  "https://www.dondominio.com",
		Fields: []FieldDefinition{
			{Name: "username", Label: "Username", Type: "text", Required: true},
			{Name: "password", Label: "Password", Type: "password", Required: false, Help: "Deprecated, use key instead"},
			{Name: "key", Label: "Key", Type: "password", Required: true},
		},
	},
	"dreamhost": {
		Name: "Dreamhost",
		URL:  "https://www.dreamhost.com",
		Fields: []FieldDefinition{
			{Name: "key", Label: "API Key", Type: "password", Required: true},
		},
	},
	"duckdns": {
		Name: "DuckDNS",
		URL:  "https://www.duckdns.org",
		Fields: []FieldDefinition{
			{Name: "token", Label: "Token", Type: "password", Required: true, Placeholder: "UUID format", Help: "Get your token from duckdns.org"},
		},
	},
	"dyn": {
		Name: "DynDNS",
		URL:  "https://dyn.com",
		Fields: []FieldDefinition{
			{Name: "username", Label: "Username", Type: "text", Required: true},
			{Name: "password", Label: "Password", Type: "password", Required: false, Help: "Deprecated, use client_key instead"},
			{Name: "client_key", Label: "Client Key", Type: "password", Required: true},
		},
	},
	"dynu": {
		Name: "Dynu",
		URL:  "https://www.dynu.com",
		Fields: []FieldDefinition{
			{Name: "username", Label: "Username", Type: "text", Required: true},
			{Name: "password", Label: "Password", Type: "password", Required: true, Help: "Can be plain text, MD5, or SHA256"},
			{Name: "group", Label: "Group", Type: "text", Required: false},
		},
	},
	"dynv6": {
		Name: "DynV6",
		URL:  "https://dynv6.com",
		Fields: []FieldDefinition{
			{Name: "token", Label: "Token", Type: "password", Required: true},
		},
	},
	"easydns": {
		Name: "EasyDNS",
		URL:  "https://www.easydns.com",
		Fields: []FieldDefinition{
			{Name: "username", Label: "Username", Type: "text", Required: true},
			{Name: "token", Label: "Token", Type: "password", Required: true},
		},
	},
	"example": {
		Name: "Example",
		URL:  "",
		Fields: []FieldDefinition{
			{Name: "username", Label: "Username", Type: "text", Required: true},
			{Name: "password", Label: "Password", Type: "password", Required: true},
		},
	},
	"freedns": {
		Name: "FreeDNS",
		URL:  "https://freedns.afraid.org",
		Fields: []FieldDefinition{
			{Name: "token", Label: "Token", Type: "password", Required: true, Help: "Enable v2 dynamic DNS at freedns.afraid.org/dynamic/v2/"},
		},
	},
	"gandi": {
		Name: "Gandi",
		URL:  "https://www.gandi.net",
		Fields: []FieldDefinition{
			{Name: "ttl", Label: "TTL", Type: "number", Required: false, Placeholder: "3600", Help: "Default: 3600"},
		},
		AuthGroups: []AuthGroup{
			{
				Name: "Personal Access Token (recommended)",
				Fields: []FieldDefinition{
					{Name: "personal_access_token", Label: "Personal Access Token", Type: "password", Required: true},
				},
			},
			{
				Name: "API Key (deprecated)",
				Fields: []FieldDefinition{
					{Name: "key", Label: "API Key", Type: "password", Required: true, Help: "Deprecated, use Personal Access Token"},
				},
			},
		},
	},
	"gcp": {
		Name: "Google Cloud Platform",
		URL:  "https://cloud.google.com",
		Fields: []FieldDefinition{
			{Name: "project", Label: "Project", Type: "text", Required: true, Placeholder: "GCP project ID"},
			{Name: "zone", Label: "Zone", Type: "text", Required: true, Placeholder: "DNS zone name"},
			{Name: "credentials", Label: "Credentials JSON", Type: "password", Required: true, Help: "Full service account JSON credentials object"},
		},
	},
	"godaddy": {
		Name: "GoDaddy",
		URL:  "https://www.godaddy.com",
		Fields: []FieldDefinition{
			{Name: "key", Label: "API Key", Type: "password", Required: true, Placeholder: "Production API key"},
			{Name: "secret", Label: "API Secret", Type: "password", Required: true},
		},
	},
	"goip": {
		Name: "GoIP.de",
		URL:  "https://www.goip.de",
		Fields: []FieldDefinition{
			{Name: "username", Label: "Username", Type: "text", Required: true},
			{Name: "password", Label: "Password", Type: "password", Required: true},
		},
	},
	"he": {
		Name: "Hurricane Electric",
		URL:  "https://dns.he.net",
		Fields: []FieldDefinition{
			{Name: "password", Label: "Password", Type: "password", Required: true},
		},
	},
	"hetzner": {
		Name: "Hetzner",
		URL:  "https://www.hetzner.com",
		Fields: []FieldDefinition{
			{Name: "zone_identifier", Label: "Zone Identifier", Type: "text", Required: true},
			{Name: "token", Label: "API Token", Type: "password", Required: true},
			{Name: "ttl", Label: "TTL", Type: "number", Required: false, Placeholder: "1", Help: "Default: 1"},
		},
	},
	"infomaniak": {
		Name: "Infomaniak",
		URL:  "https://www.infomaniak.com",
		Fields: []FieldDefinition{
			{Name: "username", Label: "DynDNS Username", Type: "text", Required: true, Help: "Use DynDNS credentials, not admin"},
			{Name: "password", Label: "DynDNS Password", Type: "password", Required: true, Help: "Use DynDNS credentials, not admin"},
		},
	},
	"inwx": {
		Name: "INWX",
		URL:  "https://www.inwx.de",
		Fields: []FieldDefinition{
			{Name: "username", Label: "Username", Type: "text", Required: true},
			{Name: "password", Label: "Password", Type: "password", Required: true},
		},
	},
	"ionos": {
		Name: "Ionos",
		URL:  "https://www.ionos.com",
		Fields: []FieldDefinition{
			{Name: "api_key", Label: "API Key", Type: "password", Required: true, Placeholder: "prefix.key", Help: "Format: prefix.key"},
		},
	},
	"linode": {
		Name: "Linode",
		URL:  "https://www.linode.com",
		Fields: []FieldDefinition{
			{Name: "token", Label: "API Token", Type: "password", Required: true},
		},
	},
	"loopia": {
		Name: "Loopia",
		URL:  "https://www.loopia.se",
		Fields: []FieldDefinition{
			{Name: "username", Label: "Username", Type: "text", Required: true},
			{Name: "password", Label: "Password", Type: "password", Required: true},
		},
	},
	"luadns": {
		Name: "LuaDNS",
		URL:  "https://www.luadns.com",
		Fields: []FieldDefinition{
			{Name: "email", Label: "Email", Type: "text", Required: true},
			{Name: "token", Label: "Token", Type: "password", Required: true},
		},
	},
	"myaddr": {
		Name: "Myaddr.tools",
		URL:  "https://myaddr.tools",
		Fields: []FieldDefinition{
			{Name: "key", Label: "Key", Type: "password", Required: true},
		},
	},
	"namecheap": {
		Name: "Namecheap",
		URL:  "https://www.namecheap.com",
		Fields: []FieldDefinition{
			{Name: "password", Label: "Dynamic DNS Password", Type: "password", Required: true, Placeholder: "32-character hex", Help: "IPv4 only"},
		},
	},
	"name.com": {
		Name: "Name.com",
		URL:  "https://www.name.com",
		Fields: []FieldDefinition{
			{Name: "username", Label: "Username", Type: "text", Required: true},
			{Name: "token", Label: "Token", Type: "password", Required: true},
			{Name: "ttl", Label: "TTL", Type: "number", Required: false, Placeholder: "300", Help: "Minimum: 300"},
		},
	},
	"namesilo": {
		Name: "NameSilo",
		URL:  "https://www.namesilo.com",
		Fields: []FieldDefinition{
			{Name: "key", Label: "API Key", Type: "password", Required: true},
			{Name: "ttl", Label: "TTL", Type: "number", Required: false, Placeholder: "7207", Help: "Range: 3600-2592000"},
		},
	},
	"netcup": {
		Name: "Netcup",
		URL:  "https://www.netcup.de",
		Fields: []FieldDefinition{
			{Name: "api_key", Label: "API Key", Type: "password", Required: true},
			{Name: "password", Label: "API Password", Type: "password", Required: true, Help: "API password, not account password"},
			{Name: "customer_number", Label: "Customer Number", Type: "text", Required: true},
		},
	},
	"njalla": {
		Name: "Njalla",
		URL:  "https://njal.la",
		Fields: []FieldDefinition{
			{Name: "key", Label: "Key", Type: "password", Required: true},
		},
	},
	"noip": {
		Name: "No-IP",
		URL:  "https://www.noip.com",
		Fields: []FieldDefinition{
			{Name: "username", Label: "Username", Type: "text", Required: true},
			{Name: "password", Label: "Password", Type: "password", Required: true},
		},
	},
	"nowdns": {
		Name: "Now-DNS",
		URL:  "https://now-dns.com",
		Fields: []FieldDefinition{
			{Name: "username", Label: "Email", Type: "text", Required: true},
			{Name: "password", Label: "Password", Type: "password", Required: true},
		},
	},
	"opendns": {
		Name: "OpenDNS",
		URL:  "https://www.opendns.com",
		Fields: []FieldDefinition{
			{Name: "username", Label: "Username", Type: "text", Required: true},
			{Name: "password", Label: "Password", Type: "password", Required: true},
		},
	},
	"ovh": {
		Name: "OVH",
		URL:  "https://www.ovh.com",
		Fields: []FieldDefinition{
			{Name: "mode", Label: "Mode", Type: "select", Required: false, Options: []string{"dynamic", "api"}, Help: "DynHost (dynamic) or ZoneDNS API (api)"},
		},
		AuthGroups: []AuthGroup{
			{
				Name: "DynHost (dynamic)",
				Fields: []FieldDefinition{
					{Name: "username", Label: "Username", Type: "text", Required: true},
					{Name: "password", Label: "Password", Type: "password", Required: true},
				},
			},
			{
				Name: "ZoneDNS API",
				Fields: []FieldDefinition{
					{Name: "api_endpoint", Label: "API Endpoint", Type: "select", Required: false, Options: []string{"ovh-eu", "ovh-ca", "ovh-us", "soyoustart-eu", "soyoustart-ca", "kimsufi-eu", "kimsufi-ca"}, Help: "Default: ovh-eu"},
					{Name: "app_key", Label: "App Key", Type: "password", Required: true},
					{Name: "app_secret", Label: "App Secret", Type: "password", Required: true},
					{Name: "consumer_key", Label: "Consumer Key", Type: "password", Required: true},
				},
			},
		},
	},
	"porkbun": {
		Name: "Porkbun",
		URL:  "https://porkbun.com",
		Fields: []FieldDefinition{
			{Name: "api_key", Label: "API Key", Type: "password", Required: true},
			{Name: "secret_api_key", Label: "Secret API Key", Type: "password", Required: true},
			{Name: "ttl", Label: "TTL", Type: "number", Required: false},
		},
	},
	"route53": {
		Name: "Route53 (AWS)",
		URL:  "https://aws.amazon.com/route53",
		Fields: []FieldDefinition{
			{Name: "access_key", Label: "Access Key", Type: "password", Required: true},
			{Name: "secret_key", Label: "Secret Key", Type: "password", Required: true},
			{Name: "zone_id", Label: "Zone ID", Type: "text", Required: true},
			{Name: "ttl", Label: "TTL", Type: "number", Required: false, Placeholder: "300", Help: "Default: 300"},
		},
	},
	"selfhost.de": {
		Name: "Selfhost.de",
		URL:  "https://www.selfhost.de",
		Fields: []FieldDefinition{
			{Name: "username", Label: "DynDNS Username", Type: "text", Required: true},
			{Name: "password", Label: "DynDNS Password", Type: "password", Required: true},
		},
	},
	"servercow": {
		Name: "Servercow",
		URL:  "https://www.servercow.de",
		Fields: []FieldDefinition{
			{Name: "username", Label: "Username", Type: "text", Required: true},
			{Name: "password", Label: "Password", Type: "password", Required: true},
			{Name: "ttl", Label: "TTL", Type: "number", Required: false, Placeholder: "120", Help: "Default: 120"},
		},
	},
	"spdyn": {
		Name: "Spdyn",
		URL:  "https://www.spdyn.de",
		Fields: []FieldDefinition{},
		AuthGroups: []AuthGroup{
			{
				Name: "Token",
				Fields: []FieldDefinition{
					{Name: "token", Label: "Token", Type: "password", Required: true},
				},
			},
			{
				Name: "User & Password",
				Fields: []FieldDefinition{
					{Name: "user", Label: "User", Type: "text", Required: true},
					{Name: "password", Label: "Password", Type: "password", Required: true},
				},
			},
		},
	},
	"strato": {
		Name: "Strato",
		URL:  "https://www.strato.de",
		Fields: []FieldDefinition{
			{Name: "password", Label: "DynDNS Password", Type: "password", Required: true},
		},
	},
	"variomedia": {
		Name: "Variomedia",
		URL:  "https://www.variomedia.de",
		Fields: []FieldDefinition{
			{Name: "email", Label: "Email", Type: "text", Required: true},
			{Name: "password", Label: "DNS Password", Type: "password", Required: true, Help: "DNS settings password, not account password"},
		},
	},
	"vultr": {
		Name: "Vultr",
		URL:  "https://www.vultr.com",
		Fields: []FieldDefinition{
			{Name: "apikey", Label: "API Key", Type: "password", Required: true},
			{Name: "ttl", Label: "TTL", Type: "number", Required: false, Placeholder: "900", Help: "Default: 900"},
		},
	},
	"zoneedit": {
		Name: "Zoneedit",
		URL:  "https://www.zoneedit.com",
		Fields: []FieldDefinition{
			{Name: "username", Label: "Username", Type: "text", Required: true},
			{Name: "token", Label: "Token", Type: "password", Required: true},
		},
	},
}
