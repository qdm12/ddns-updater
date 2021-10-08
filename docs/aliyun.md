# Aliyun

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "aliyun",
      "domain": "domain.com",
      "host": "@",
      "access_key_id": "your access_key_id",
      "access_secret": "your access_secret",
      "ip_version": "ipv4"
    }
  ]
}
```

### Compulsory parameters

- `"domain"`
- `"host"` is your host and can be a subdomain or `"@"`
- `"access_key_id"`
- `"access_secret"`

### Optional parameters

- `"ip_version"` can be `ipv4` (A records) or `ipv6` (AAAA records), defaults to `ipv4 or ipv6`

## Domain setup
