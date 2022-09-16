# Gandi

This provider uses Gandi v5 API

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "gandi",
      "domain": "domain.com",
      "host": "@",
      "key": "key",
      "ttl": 3600,
      "ip_version": "ipv4"
    }
  ]
}
```

### Compulsory parameters

- `"domain"`
- `"host"` which can be a subdomain, `@` or a wildcard `*`
- `"key"`

### Optional parameters

- `"ip_version"` can be `ipv4` (A records) or `ipv6` (AAAA records), defaults to `ipv4 or ipv6`
- `"ttl"` default is `3600`

## Domain setup

[Gandi Documentation Website](https://docs.gandi.net/en/domain_names/advanced_users/api.html#gandi-s-api)
