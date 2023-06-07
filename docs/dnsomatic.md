# DNS-O-Matic

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "dnsomatic",
      "domain": "domain.com",
      "host": "@",
      "username": "username",
      "password": "password",
      "provider_ip": true,
      "ip_version": "ipv4"
    }
  ]
}
```

### Compulsory parameters

- `"domain"`
- `"host"` is your host and can be a subdomain or `"@"` or `"*"`
- `"username"`
- `"password"`
- `"provider_ip"`

### Optional parameters

- `"ip_version"` can be `ipv4` (A records) or `ipv6` (AAAA records), defaults to `ipv4 or ipv6`

## Domain setup
