# deSEC

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "desec",
      "domain": "dedyn.io",
      "host": "host",
      "token": "token",
      "ip_version": "ipv4",
      "provider_ip": false
    }
  ]
}
```

### Compulsory parameters

- `"domain"`
- `"host"`
- `"token"` is your token that you can create [here](https://desec.io/tokens)

### Optional parameters

- `"ip_version"` can be `ipv4` (A records) or `ipv6` (AAAA records), defaults to `ipv4 or ipv6`
- `"provider_ip"` can be set to `true` to let your DNS provider determine your IPv4 address (and/or IPv6 address) automatically when you send an update request, without sending the new IP address detected by the program in the request.

## Domain setup

### Web

[desec.io/domains](https://desec.io/domains)

### API

[desec.readthedocs.io/en/latest/dns/domains](https://desec.readthedocs.io/en/latest/dns/domains.html)
