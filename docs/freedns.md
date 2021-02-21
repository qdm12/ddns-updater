# FreeDNS

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "freedns",
      "domain": "domain.com",
      "host": "host",
      "id": "password",
      "ip_version": "ipv4"
    }
  ]
}
```

### Compulsory parameters

- `"domain"`
- `"host"` is your host (subdomain)
- `"id"` is the ID you use to update your record

### Optional parameters

- `"ip_version"` can be `ipv4` (A records) or `ipv6` (AAAA records), defaults to `ipv4 or ipv6`

## Domain setup
