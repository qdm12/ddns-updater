# Name.com

<img src="../readme/name.svg" alt="drawing" width="25%"/>

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "name.com",
      "domain": "domain.com",
      "username": "username",
      "token": "token",
      "ttl": 300,
      "ip_version": "ipv4",
      "ipv6_suffix": ""
    }
  ]
}
```

### Compulsory parameters

- `"domain"` is the domain to update. It can be `example.com` (root domain), `sub.example.com` (subdomain of `example.com`) or `*.example.com` for the wildcard.
- `"username"` is your account username
- `"token"` which you can obtain from [www.name.com/account/settings/api](https://www.name.com/account/settings/api)

### Optional parameters

- `"ttl"` is the time this record can be cached for in seconds. Name.com allows a minimum TTL of 300, or 5 minutes. Name.com defaults to 300 if not provided.
- `"ip_version"` can be `ipv4` (A records), or `ipv6` (AAAA records) or `ipv4 or ipv6` (update one of the two, depending on the public ip found). It defaults to `ipv4 or ipv6`.
- `"ipv6_suffix"` is the IPv6 interface identifier suffix to use. It can be for example `0:0:0:0:72ad:8fbb:a54e:bedd/64`. If left empty, it defaults to no suffix and the raw public IPv6 address obtained is used in the record updating.
