# All-Inkl

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "allinkl",
      "domain": "domain.com",
      "host": "host",
      "username": "dynXXXXXXX",
      "password": "password",
      "ip_version": "ipv4"
    }
  ]
}
```

### Compulsory parameters

- `"domain"`
- `"host"` is your host (subdomain)
- `"username"` username (usually starts with dyn followed by numbers)
- `"password"` password in plain text

### Optional parameters

- `"ip_version"` can be `ipv4` (A records), or `ipv6` (AAAA records) or `ipv4 or ipv6` (update one of the two, depending on the public ip found). It defaults to `ipv4 or ipv6`.

## Domain setup
