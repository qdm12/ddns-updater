# Selfhost.de

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "selfhost.de",
      "domain": "domain.com",
      "host": "@",
      "username": "username",
      "password": "password",
      "ip_version": "ipv4"
    }
  ]
}
```

### Compulsory parameters

- `"domain"`
- `"host"` is your host and can be a subdomain or `"@"`
- `"username"` is your DynDNS username
- `"password"` is your DynDNS password

### Optional parameters

- `"ip_version"` can be `ipv4` (A records) or `ipv6` (AAAA records), defaults to `ipv4 or ipv6`

## Domain setup
