# Spaceship

## Configuration

Docs can be found for the spaceship API [here](https://docs.spaceship.dev/).
### Example

```json
{
  "settings": [
    {
      "provider": "spaceship",
      "domain": "example.com",
      "host": "subdomain",
      "apikey": "YOUR_API_KEY",
      "apisecret": "YOUR_API_SECRET",
      "ip_version": "ipv4"
    }
  ]
}
```

### Compulsory parameters

- `"domain"` is the domain to update. It can be a root domain (i.e. `example.com`) or a subdomain (i.e. `subdomain.example.com`).
- `"apikey"` is your API key which can be obtained from [API Manager](https://www.spaceship.com/application/api-manager/).
- `"apisecret"` is your API secret which is provided along with your API key in the API Manager.

### Optional parameters

- `"ip_version"` can be `ipv4` (A records), or `ipv6` (AAAA records) or `ipv4 or ipv6` (update one of the two, depending on the public ip found). It defaults to `ipv4 or ipv6`.
- `"ipv6_suffix"` is the IPv6 interface identifier suffix to use. It can be for example `0:0:0:0:72ad:8fbb:a54e:bedd/64`. If left empty, it defaults to no suffix and the raw public IPv6 address obtained is used in the record updating.
- `"ttl"` is the record TTL which defaults to 3600 seconds.
