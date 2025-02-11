# INWX

## Configuration

You must configure a DynDNS Account in INWX and use the configured account and hostname. See https://www.inwx.com/en/offer/dyndns.

### Example

```json
{
  "settings": [
    {
      "provider": "inwx",
      "domain": "domain.com",
      "username": "username",
      "password": "password",
      "ip_version": "ipv4",
      "ipv6_suffix": ""
    }
  ]
}
```

### Compulsory parameters

- `"domain"` is the domain to update. This must be the domain configured for the DynDNS Account.
- `"username"`
- `"password"`

### Optional parameters

- `"ip_version"` can be `ipv4` (A records), or `ipv6` (AAAA records) or `ipv4 or ipv6` (update one of the two, depending on the public ip found). It defaults to `ipv4 or ipv6`.
- `"ipv6_suffix"` is the IPv6 interface identifier suffix to use. It can be for example `0:0:0:0:72ad:8fbb:a54e:bedd/64`. If left empty, it defaults to no suffix and the raw public IPv6 address obtained is used in the record updating.

## Domain setup
