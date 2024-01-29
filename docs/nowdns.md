# Now-DNS

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "nowdns",
      "domain": "domain.com",
      "username": "username",
      "password": "password",
      "ip_version": "ipv4"
    }
  ]
}
```

### Compulsory parameters

- `"domain"` your full domain name (FQDN)
- `"username"` your email address
- `"password"`

### Optional parameters

- `"ip_version"` can be `ipv4` (A records), or `ipv6` (AAAA records) or `ipv4 or ipv6` (update one of the two, depending on the public ip found). It defaults to `ipv4 or ipv6`.
