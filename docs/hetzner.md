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
      "ttl": 600,
      "token": "yourtoken",
      "ip_version": "ipv4",
      "ipv6_suffix": ""
    }
  ]
}
```

### Compulsory parameters

- `"zone_identifier"` is the Zone ID of your site, from the domain overview page written as *Zone ID*
- `"domain"` is the domain to update. It can be `example.com` (root domain), `sub.example.com` (subdomain of `example.com`) or `*.example.com` for the wildcard.
- `"ttl"` optional integer value corresponding to a number of seconds
- One of the following ([how to find API keys](https://docs.hetzner.com/cloud/api/getting-started/generating-api-token)):
  - API Token `"token"`, configured with DNS edit permissions for your DNS name's zone

### Optional parameters

- `"ip_version"` can be `ipv4` (A records), or `ipv6` (AAAA records) or `ipv4 or ipv6` (update one of the two, depending on the public ip found). It defaults to `ipv4 or ipv6`.
- `"ipv6_suffix"` is the IPv6 interface identifier suffix to use. It can be for example `0:0:0:0:72ad:8fbb:a54e:bedd/64`. If left empty, it defaults to no suffix and the raw public IPv6 address obtained is used in the record updating.
