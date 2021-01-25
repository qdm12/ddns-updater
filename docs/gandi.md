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
      "name": "@",
      "key": "key",
      "ttl": 3600,
      "ip_version": "ipv4",
    }
  ]
}
```

If no ttl is defined, default is 3600

### Compulsory parameters

- `"domain"` is your fqdn, for example `subdomain.duckdns.org`
- `"key"`

## Domain setup

[Gandi Documentation Website](https://docs.gandi.net/en/domain_names/advanced_users/api.html#gandi-s-api)
