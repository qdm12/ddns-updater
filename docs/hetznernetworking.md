# Hetzner Networking

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "hetznernetworking",
      "domain": "example.com",
      "token": "yourtoken",
      "ip_version": "ipv4",
      "ipv6_suffix": ""
    }
  ]
}
```

### Compulsory parameters

- `"domain"` is the domain to update. It can be `example.com` (root domain), `sub.example.com` (subdomain of `example.com`) or `*.example.com` for the wildcard.
- One of the following ([how to find API keys](https://docs.hetzner.cloud/api/getting-started/generating-api-token)):
  - API Token `"token"`, configured with DNS write permissions for your DNS zone

### Optional parameters

- `"ip_version"` can be `ipv4` (A records), or `ipv6` (AAAA records) or `ipv4 or ipv6` (update one of the two, depending on the public ip found). It defaults to `ipv4 or ipv6`.
- `"ipv6_suffix"` is the IPv6 interface identifier suffix to use. It can be for example `0:0:0:0:72ad:8fbb:a54e:bedd/64`. If left empty, it defaults to no suffix and the raw public IPv6 address obtained is used in the record updating.
- `"ttl"` time to live for the DNS record in seconds. It is only used to add a record to the rrset, and is not used to update an existing record. If left empty, it defaults to the existing zone TTL.

## Notes

This provider uses the Hetzner Networking DNS API (<https://api.hetzner.cloud/v1/>) which is different from the legacy Hetzner DNS API.

- For subdomains, the provider automatically extracts the subdomain part relative to the zone
- For apex records (root domain), the provider uses `@` as the record name
- The API uses RRSet-based operations for managing DNS records

For more information about the Hetzner Networking DNS API, see the [official documentation](https://docs.hetzner.cloud/reference/cloud#dns).
