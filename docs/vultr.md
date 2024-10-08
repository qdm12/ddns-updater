# Vultr

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "vultr",
      "domain": "potato.example.com",
      "token": "AAAAAAAAAAAAAAA",
      "ttl": 300,
      "ip_version": "ipv4"
    }
  ]
}
```

### Compulsory parameters

- `"domain"` is the domain to update. It can be `example.com` (root domain) or `potato.example.com` (subdomain of `example.com`).
- `"token"` is your API key, can be obtained from the [account settings](https://my.vultr.com/settings/#settingsapi), this is used as a bearer token to authenticate your request

### Optional parameters

- `"ip_version"` can be `ipv4` (A records), or `ipv6` (AAAA records) or `ipv4 or ipv6` (update one of the two, depending on the public ip found). It defaults to `ipv4 or ipv6`.
- `"ipv6_suffix"` is the IPv6 interface identifier suffix to use. It can be for example `0:0:0:0:72ad:8fbb:a54e:bedd/64`. If left empty, it defaults to no suffix and the raw public IPv6 address obtained is used in the record updating.
- `"ttl"` default is 900