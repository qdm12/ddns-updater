# Hetzner Cloud

Uses the [Hetzner Cloud API](https://docs.hetzner.cloud/reference/cloud) (`api.hetzner.cloud/v1`) which replaced the legacy Hetzner DNS API (`dns.hetzner.com`) in 2026.

> **Migration note:** If you are currently using `"provider": "hetzner"` (legacy DNS API), migrate to `"provider": "hetznercloud"` before May 2026 when the old API is shut down.

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "hetznercloud",
      "domain": "domain.com",
      "token": "yourtoken",
      "ip_version": "ipv4",
      "ipv6_suffix": ""
    }
  ]
}
```

With explicit zone ID (optional, saves one API lookup per update):

```json
{
  "settings": [
    {
      "provider": "hetznercloud",
      "zone_identifier": "your-zone-id",
      "domain": "domain.com",
      "ttl": 3600,
      "token": "yourtoken",
      "ip_version": "ipv4",
      "ipv6_suffix": ""
    }
  ]
}
```

### Compulsory parameters

- `"domain"` is the domain to update. It can be `example.com` (root domain), `sub.example.com` (subdomain of `example.com`) or `*.example.com` for the wildcard.
- `"token"` is a Hetzner Cloud API token with DNS read and write permissions. [How to generate an API token](https://docs.hetzner.com/cloud/api/getting-started/generating-api-token)

### Optional parameters

- `"zone_identifier"` is the Zone ID of your DNS zone from the Hetzner Cloud Console. If omitted, it is looked up automatically from `domain`.
- `"ttl"` is the TTL in seconds for the DNS record. Defaults to `3600`.
- `"ip_version"` can be `ipv4` (A records), `ipv6` (AAAA records) or `ipv4 or ipv6` (updates whichever public IP is found). Defaults to `ipv4 or ipv6`.
- `"ipv6_suffix"` is the IPv6 interface identifier suffix to use, for example `0:0:0:0:72ad:8fbb:a54e:bedd/64`. Defaults to no suffix (raw public IPv6 address).
