# Hetzner

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "hetzner",
      "zone_identifier": "some id",
      "domain": "domain.com",
      "host": "@",
      "ttl": 600,
      "token": "yourtoken",
      "ip_version": "ipv4"
    }
  ]
}
```

### Compulsory parameters

- `"zone_identifier"` is the Zone ID of your site, from the domain overview page written as *Zone ID*
- `"domain"`
- `"host"` is your host and can be `"@"`, a subdomain or the wildcard `"*"`.
- `"ttl"` integer value for record TTL in seconds (specify 1 for automatic)
- One of the following ([how to find API keys](https://docs.hetzner.com/cloud/api/getting-started/generating-api-token)):
  - API Token `"token"`, configured with DNS edit permissions for your DNS name's zone

### Optional parameters

- `"ip_version"` can be `ipv4` (A records) or `ipv6` (AAAA records), and defaults to `ipv4 or ipv6`
