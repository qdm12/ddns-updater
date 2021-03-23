# Spdyn.de

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "spdyn",
      "domain": "domain.com",
      "host": "@",
      "user": "user",
      "password": "password",
      "token": "token",
      "ip_version": "ipv4",
      "provider_ip": true
    }
  ]
}
```

### Compulsory parameters

- `"domain"`
- `"host"` is your host and can be a subdomain or `"@"`

#### Using user and password

- `"user"` is the name of a user who can update this host
- `"password"` is the password of a user who can update this host

#### Using update tokens

- `"token"` is your update token

### Optional parameters

- `"ip_version"` can be `ipv4` (A records) or `ipv6` (AAAA records), defaults to `ipv4 or ipv6`
- `"provider_ip"` can be set to `true` to let your DNS provider determine your IPv4 address (**not IPv6**)automatically when you send an update request, without sending the new IP address detected by the program in the request.
