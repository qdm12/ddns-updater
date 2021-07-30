# Google

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "google",
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
- `"host"` is your host and can be a subdomain or `"@"` or `"*"`
- `"username"`
- `"password"`

### Optional parameters

- `"ip_version"` can be `ipv4` (A records) or `ipv6` (AAAA records), defaults to `ipv4 or ipv6`

## Domain setup

Thanks to [@gauravspatel](https://github.com/gauravspatel) for #124

1. Enable dynamic DNS in the *synthetic records* section of DNS management.
1. The username and password is generated once you create the dynamic DNS entry.

### Wildcard entries

If you want to create a **wildcard entry**, you have to create a custom **CNAME** record with key `"*"` and value `"@"`.
