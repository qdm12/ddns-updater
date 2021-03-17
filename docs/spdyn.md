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
      "ip_version": "ipv4"
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
