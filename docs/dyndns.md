# DynDNS

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "dyn",
      "domain": "domain.com",
      "host": "@",
      "username": "username",
      "client_key": "client_key",
      "ip_version": "ipv4"
    }
  ]
}
```

### Compulsory parameters

- `"domain"`
- `"host"` is your host and can be a subdomain or `"@"` or `"*"`
- `"username"`
- `"client_key"`

### Optional parameters

- `"ip_version"` can be `ipv4` (A records) or `ipv6` (AAAA records), defaults to `ipv4 or ipv6`

## Domain setup
