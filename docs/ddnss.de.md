# DDNSS

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "ddnss",
      "provider_ip": true,
      "domain": "domain.com",
      "host": "@",
      "username": "user",
      "password": "password",
      "dual_stack": false,
      "ip_version": "ipv4"
    }
  ]
}
```

### Compulsory parameters

- `"domain"`
- `"host"` is your host and can be a subdomain or `"@"`
- `"username"`
- `"password"`

### Optional parameters

- `"dual_stack"` can be set to `true` **if you have turn on dual stack for your record** to update both IPv4 and IPv6 addresses. Note it is ignored if `"provider_ip": true`. More precisely:
  - if it is `false`, the updates are done using the `ip` parameter and only one IP address can be set (ipv4 or ipv6, whichever is last sent).
  - if it is `true`, the updates are done using the `ip` and `ip6` parameters, for IPv4 and IPv6 respectively, and both can be set on the same record
- `"ip_version"` can be `ipv4` (A records) or `ipv6` (AAAA records), defaults to `ipv4 or ipv6`

## Domain setup
