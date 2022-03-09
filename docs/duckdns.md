# DuckDNS

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "duckdns",
      "host": "host",
      "token": "token",
      "ip_version": "ipv4",
      "provider_ip": true
    }
  ]
}
```

### Compulsory parameters

- `"host"` is your host, for example `subdomain` for `subdomain.duckdns.org`
- `"token"`

### Optional parameters

- `"ip_version"` can be `ipv4` (A records) or `ipv6` (AAAA records), defaults to `ipv4 or ipv6`
- `"provider_ip"` can be set to `true` to let your DNS provider determine your IPv4 address (**NOT** your IPv6 address) automatically when you send an update request, without sending the new IP address detected by the program in the request.

## Domain setup

[![DuckDNS Website](../readme/duckdns.png)](https://duckdns.org)

*See the [duckdns website](https://duckdns.org)*
