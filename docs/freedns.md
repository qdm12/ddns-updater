# FreeDNS

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "freedns",
      "domain": "sub.domain.com",
      "token": "token",
      "ip_version": "ipv4",
      "ipv6_suffix": ""
    }
  ]
}
```

### Compulsory parameters

- `"domain"` is the domain to update. It can be `example.com` (root domain) or `sub.example.com` (subdomain of `example.com`).
- `"token"` is the randomized update token you use to update your record

### Optional parameters

- `"ip_version"` can be `ipv4` (A records), or `ipv6` (AAAA records) or `ipv4 or ipv6` (update one of the two, depending on the public ip found). It defaults to `ipv4 or ipv6`.
- `"ipv6_suffix"` is the IPv6 interface identifier suffix to use. It can be for example `0:0:0:0:72ad:8fbb:a54e:bedd/64`. If left empty, it defaults to no suffix and the raw public IPv6 address obtained is used in the record updating.

## Domain setup

This integration uses FreeDNS's v2 dynamic dns interface, which is not shown by default when you select `Dynamic DNS` from the side menu.
Instead you must go to [freedns.afraid.org/dynamic/v2/](https://freedns.afraid.org/dynamic/v2/) and enable dynamic DNS for the subdomains you wish and you will then see a url like `https://sync.afraid.org/u/token/` for each enabled subdomain.
