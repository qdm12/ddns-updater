# Bunny

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "bunny",
      "domain": "sub.example.com",
      "api_key": "YOUR_API_KEY",
      "ip_version": "ipv4",
      "ipv6_suffix": ""
    }
  ]
}
```

### Compulsory parameters

- `"domain"` is the domain to update. It can be `example.com` (root domain), `sub.example.com` (subdomain of `example.com`) or `*.example.com` for the wildcard.
- `"api_key"` is your API key which can be obtained from your [Account Settings](https://dash.bunny.net/account/api-key).

### Optional parameters

- `"ip_version"` can be `ipv4` (A records), or `ipv6` (AAAA records) or `ipv4 or ipv6` (update one of the two, depending on the public ip found). It defaults to `ipv4 or ipv6`.
- `"ipv6_suffix"` is the IPv6 interface identifier suffix to use. It can be for example `0:0:0:0:72ad:8fbb:a54e:bedd/64`. If left empty, it defaults to no suffix and the raw temporary IPv6 address of the machine is used in the record updating. You might want to set this to use your permanent IPv6 address instead of your temporary IPv6 address.
- `"ttl"` is the record TTL in seconds, between `60` and `3600`. Defaults to the zone default when unset.

## Domain setup

More information at the [Bunny API](https://docs.bunny.net/).
