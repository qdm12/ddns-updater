# FreeDNS

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "freedns",
      "domain": "domain.com",
      "host": "host",
      "token": "token",
      "ip_version": "ipv4"
    }
  ]
}
```

### Compulsory parameters

- `"domain"`
- `"host"` is your host (subdomain)
- `"token"` is the randomized update token you use to update your record

### Optional parameters

- `"ip_version"` can be `ipv4` (A records) or `ipv6` (AAAA records), defaults to `ipv4 or ipv6`

## Domain setup

This integration uses FreeDNS's v2 dynamic dns interface, which is not shown by default when you select `Dynamic DNS` from the side menu. Instead you must go to https://freedns.afraid.org/dynamic/v2/ and enable dynamic DNS for the subdomains you wish and you will then see a url like `https://sync.afraid.org/u/token/` for each enabled subdomain.
