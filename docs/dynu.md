# Dynu

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "dynu",
      "domain": "domain.com",
      "host": "@",
      "group": "group",
      "username": "username",
      "password": "password",
      "ip_version": "ipv4",
      "provider_ip": true
    }
  ]
}
```

### Compulsory parameters

- `"domain"`
- `"host"` is your host and can be a subdomain or `"@"`
- `"username"`
- `"password"` could be plain text or password in MD5 or SHA256 format (There's also an option for setting a password for IP Update only)

### Optional parameters

- `"ip_version"` can be `ipv4` (A records) or `ipv6` (AAAA records), defaults to `ipv4 or ipv6`
- `"provider_ip"` can be set to `true` to let your DNS provider determine your IPv4 address (and/or IPv6 address) automatically when you send an update request, without sending the new IP address detected by the program in the request.
- `"group"` specify the Group for which you want to set the IP (will update any domains and subdomains in the same group)

## Domain setup
