# DuckDNS

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "duckdns",
      "domain": "sub.duckdns.org",
      "token": "token",
      "ip_version": "ipv4",
      "ipv6_suffix": ""
    }
  ]
}
```

### Compulsory parameters

- `"domain"` is the domain to update. The [eTLD](https://developer.mozilla.org/en-US/docs/Glossary/eTLD) must be `duckdns.org`. For example:
  - for the root owner/host `@`, it would be `mydomain.duckdns.org`
  - for the owner/host `sub`, it would be `sub.mydomain.duckdns.org`
  - for multiple domains, it can be `sub1.mydomain.duckdns.org,sub2.mydomain.duckdns.org` BUT it cannot be `a.duckdns.org,b.duckdns.org`, since the effective domains would be `a.duckdns.org` and `b.duckdns.org`
- `"token"`

### Optional parameters

- `"ip_version"` can be `ipv4` (A records), or `ipv6` (AAAA records) or `ipv4 or ipv6` (update one of the two, depending on the public ip found). It defaults to `ipv4 or ipv6`.
- `"ipv6_suffix"` is the IPv6 interface identifier suffix to use. It can be for example `0:0:0:0:72ad:8fbb:a54e:bedd/64`. If left empty, it defaults to no suffix and the raw public IPv6 address obtained is used in the record updating.

## Domain setup

[![DuckDNS Website](../readme/duckdns.png)](https://www.duckdns.org/)

*See the [duckdns website](https://www.duckdns.org/)*
