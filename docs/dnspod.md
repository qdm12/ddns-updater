# DNSPod

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "dnspod",
      "domain": "domain.com",
      "host": "@",
      "token": "yourtoken",
      "ip_version": "ipv4"
    }
  ]
}
```

### Compulsory parameters

- `"domain"`
- `"host"` is your host and can be a subdomain or `"@"`
- `"token"`

### Optional parameters

- `"ip_version"` can be `ipv4` (A records), or `ipv6` (AAAA records) or `ipv4 or ipv6` (update one of the two, depending on the public ip found). It defaults to `ipv4 or ipv6`.

## Domain setup
