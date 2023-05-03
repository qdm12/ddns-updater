# Servercow

## Configuration

### Example

```json
{
  "settings": [
      {
          "provider": "servercow",
          "domain": "domain.com",
          "host": "",
          "username": "servercow_username",
          "password": "servercow_password",
          "ttl": 600,
          "provider_ip": true,
          "ip_version": "ipv4"
      }
  ]
}
```

### Compulsury parameters

- `"domain"`
- `"host"` is your host and can be `""`, a subdomain or `"*"` generally
- `"username"` is the username for your DNS API User
- `"password"` is the password for your DNS API User
- `"provider_ip"`

### Optional parameters

- `"ttl"` can be set to an integer value for record TTL in seconds (if not set the default is 120)
- `"ip_version"` can be `ipv4` (A records) or `ipv6` (AAAA records), and defaults to `ipv4 or ipv6`

## Domain setup

See [their article](https://cp.servercow.de/en/plugin/support_manager/knowledgebase/view/34/dns-api-v1/7/)
