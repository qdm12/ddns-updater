# Site.nl

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "sitenl",
      "domain": "domain.nl",
      "owner": "@",
      "api_key": "yourkey256charslong...",
      "ttl": 3600,
      "ip_version": "ipv4",
      "ipv6_suffix": ""
    }
  ]
}
```

### Compulsory parameters

- `"domain"` is the domain to update. It can be `example.nl` (root domain) or `sub.example.nl` (subdomain of `example.nl`).
- `"api_key"` is your site.nl API key. It must be exactly **256 characters** long.

### Optional parameters

- `"ttl"` is the DNS record TTL in seconds. Allowed values: `60`, `300`, `3600`, `7200`, `14400`, `28800`, `86400`. Defaults to `3600`.
- `"ip_version"` can be `ipv4` (A records), or `ipv6` (AAAA records) or `ipv4 or ipv6` (update one of the two, depending on the public ip found). It defaults to `ipv4 or ipv6`.
- `"ipv6_suffix"` is the IPv6 interface identifier suffix to use. It can be for example `0:0:0:0:72ad:8fbb:a54e:bedd/64`. If left empty, it defaults to no suffix and the raw temporary IPv6 address of the machine is used in the record updating. You might want to set this to use your permanent IPv6 address instead of your temporary IPv6 address.

## Domain setup

1. Log in to your [site.nl control panel](https://www.site.nl/).
2. Navigate to your domain's DNS settings and make sure **DNS control** is enabled for the domain you want to update.
3. Go to **Account settings → API keys** and create a new API key with at minimum the following permissions:
   - `domains.view`
   - `domains.dns.modify`
4. Copy the generated 256-character API key and use it as the `"api_key"` value in your configuration.

> ⚠️ Site.nl API keys are exactly 256 characters. After 3 failed authentication attempts within 24 hours your IP will be blocked for 72 hours — store your key securely.

## Record behaviour

The site.nl API (`PATCH /v2/domain_names/{id}/dns_records`) performs a **complete replacement** of all DNS records on each update. DDNS-Updater fetches the current record set first, upserts the A/AAAA record for the configured owner, and then sends the full set back — so all your other DNS records are preserved.
