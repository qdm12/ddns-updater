# Strato

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "strato",
      "domain": "domain.com",
      "host": "@",
      "password": "password",
      "ip_version": "ipv4",
      "provider_ip": true
    }
  ]
}
```

### Compulsory parameters

- `"domain"`
- `"host"` is your host and can be a subdomain or `"@"`
- `"password"` is your dyndns password

### Optional parameters

- `"ip_version"` can be `ipv4` (A records) or `ipv6` (AAAA records), defaults to `ipv4 or ipv6`
- `"provider_ip"` can be set to `true` to let your DNS provider determine your IPv4 address (and/or IPv6 address) automatically when you send an update request, without sending the new IP address detected by the program in the request.

## Domain setup

See [their article](https://www.strato.com/faq/en_us/domain/this-is-how-easy-it-is-to-set-up-dyndns-for-your-domains/)
