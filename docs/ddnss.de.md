# DDNSS

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "ddnss",
      "provider_ip": true,
      "domain": "domain.com",
      "host": "@",
      "user": "user",
      "password": "password",
      "ip_version": "ipv4"
    }
  ]
}
```

### Compulsory parameters

- `"domain"`
- `"host"` is your host and can be a subdomain or `"@"`
- `"user"`
- `"password"`

### Optional parameters

- `"ip_version"` can be `ipv4` (A records) or `ipv6` (AAAA records), defaults to `ipv4 or ipv6`

## Domain setup
