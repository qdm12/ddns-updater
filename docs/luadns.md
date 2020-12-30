# LuaDNS

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "luadns",
      "domain": "domain.com",
      "host": "@",
      "email": "email",
      "token": "token",
      "ip_version": "ipv4"
    }
  ]
}
```

### Compulsory parameters

- `"domain"`
- `"host"` is your host and can be a subdomain or `"@"` or `"*"`
- `"email"`
- `"token"`

### Optional parameters

- `"ip_version"` can be `ipv4` (A records) or `ipv6` (AAAA records), defaults to `ipv4 or ipv6`

## Domain setup

1. Go to [api.luadns.com/settings](https://api.luadns.com/settings)
1. Enable API access
1. Obtain your API token and replace it in the parameters as the value for `token`
