# Vultr

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "vultr",
      "domain": "potato.example.com",
      "apikey": "AAAAAAAAAAAAAAA",
      "ttl": 300,
      "ip_version": "ipv4"
    }
  ]
}
```

### Compulsory parameters

- `"domain"` is the domain to update. It can be a root domain (i.e. `example.com`) or a subdomain (i.e. `potato.example.com`) or a wildcard (i.e. `*.example.com`). In case of a wildcard, it only works if there is no existing wildcard records of any record type.
- `"apikey"` is your API key which can be obtained from [my.vultr.com/settings/](https://my.vultr.com/settings/#settingsapi).

### Optional parameters

- `"ip_version"` can be `ipv4` (A records), or `ipv6` (AAAA records) or `ipv4 or ipv6` (update one of the two, depending on the public ip found). It defaults to `ipv4 or ipv6`.
- `"ipv6_suffix"` is the IPv6 interface identifier suffix to use. It can be for example `0:0:0:0:72ad:8fbb:a54e:bedd/64`. If left empty, it defaults to no suffix and the raw public IPv6 address obtained is used in the record updating.
- `"ttl"` is the record TTL which defaults to 900 seconds.
