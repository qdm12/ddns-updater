# Dreamhost

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "dreamhost",
      "domain": "domain.com",
      "key": "key",
      "ip_version": "ipv4"
    }
  ]
}
```

### Compulsory parameters

- `"domain"`
- `"key"`

### Optional parameters

- `"ip_version"` can be `ipv4` (A records) or `ipv6` (AAAA records), defaults to `ipv4 or ipv6`

## Domain setup
