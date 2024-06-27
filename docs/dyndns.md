# DynDNS

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "dyn",
      "domain": "domain.com",
      "owner": "@",
      "username": "username",
      "client_key": "client_key",
      "ip_version": "ipv4",
      "ipv6_suffix": ""
    }
  ]
}
```

### Compulsory parameters

- `"domain"`
- `"owner"` is the resource record owner and can be `"@"` for the root domain, `"*"` for the wildcard, or for example `"sub"` for the subdomain `sub.domain.com`
- `"username"`
- `"client_key"`

### Optional parameters

- `"ip_version"` can be `ipv4` (A records), or `ipv6` (AAAA records) or `ipv4 or ipv6` (update one of the two, depending on the public ip found). It defaults to `ipv4 or ipv6`.
- `"ipv6_suffix"` is the IPv6 interface identifier suffix to use. It can be for example `0:0:0:0:72ad:8fbb:a54e:bedd/64`. If left empty, it defaults to no suffix and the raw public IPv6 address obtained is used in the record updating.

## Domain setup
