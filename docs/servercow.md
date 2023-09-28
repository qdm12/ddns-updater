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

### Compulsory parameters

- `"domain"`
- `"host"` is your host and can be `""`, a subdomain or `"*"` generally
- `"username"` is the username for your DNS API User
- `"password"` is the password for your DNS API User

### Optional parameters

- `"ttl"` can be set to an integer value for record TTL in seconds (if not set the default is 120)
- `"ip_version"` can be `ipv4` (A records) or `ipv6` (AAAA records), and defaults to `ipv4 or ipv6`
- `"provider_ip"` can be set to `true` to let your DNS provider determine your IPv4 address (and/or IPv6 address) automatically when you send an update request, without sending the new IP address detected by the program in the request.

## Domain setup

See [their article](https://cp.servercow.de/en/plugin/support_manager/knowledgebase/view/34/dns-api-v1/7/)
