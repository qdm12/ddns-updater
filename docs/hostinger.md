# Hostinger

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "hostinger",
      "domain": "domain.com",
      "token": "YOUR_HOSTINGER_API_TOKEN",
      "ttl": 300,
      "ip_version": "ipv4",
      "ipv6_suffix": ""
    }
  ]
}
```

### Compulsory parameters

- `"domain"` is the domain to update. It can be `example.com` for the root
  domain, `sub.example.com` for a subdomain or `*.example.com` for a wildcard.
- `"token"` is a Hostinger API token. Tokens can be created and managed from
  the [Hostinger API page](https://hpanel.hostinger.com/profile/api).

### Optional parameters

- `"ttl"` is the DNS record time to live in seconds. It defaults to `14400`.
- `"ip_version"` can be `ipv4` for A records, `ipv6` for AAAA records or
  `ipv4 or ipv6` to update one of the two depending on the public IP found. It
  defaults to `ipv4 or ipv6`.
- `"ipv6_suffix"` is the IPv6 interface identifier suffix to use. It can be,
  for example, `0:0:0:0:72ad:8fbb:a54e:bedd/64`. If left empty, no suffix is
  used and the raw temporary public IPv6 address is used for the update. Set
  this value to use a permanent IPv6 address instead.
