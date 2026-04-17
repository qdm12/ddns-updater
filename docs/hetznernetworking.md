# Hetzner Networking

This provider uses the Hetzner Cloud API `https://api.hetzner.cloud/v1/` which is different from the legacy Hetzner DNS API.

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
- `"token"` is your API token configured with DNS write permissions for your DNS zone, see [the Authentication section](https://docs.hetzner.cloud/reference/cloud#description/authentication)

### Optional parameters

- `"ip_version"` can be `ipv4` (A records), or `ipv6` (AAAA records) or `ipv4 or ipv6` (update one of the two, depending on the public ip found). It defaults to `ipv4 or ipv6`.
- `"ipv6_suffix"` is the IPv6 interface identifier suffix to use. It can be for example `0:0:0:0:72ad:8fbb:a54e:bedd/64`. If left empty, it defaults to no suffix and the raw temporary IPv6 address of the machine is used in the record updating. You might want to set this to use your permanent IPv6 address instead of your temporary IPv6 address.
- `"ttl"` time to live for the DNS record in seconds. It is only used to add a record to the rrset, and is not used to update an existing record. If left empty, it defaults to the existing zone TTL.
