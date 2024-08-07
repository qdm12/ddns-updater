# Servercow

## Configuration

### Example

```json
{
  "settings": [
      {
          "provider": "servercow",
          "domain": "domain.com",
          "username": "servercow_username",
          "password": "servercow_password",
          "ttl": 600,
          "ip_version": "ipv4",
          "ipv6_suffix": ""
      }
  ]
}
```

### Compulsory parameters

- `"domain"` is the domain to update. It can be `example.com` (root domain), `sub.example.com` (subdomain of `example.com`) or `*.example.com` for the wildcard.
- `"username"` is the username for your DNS API User
- `"password"` is the password for your DNS API User

### Optional parameters

- `"ttl"` can be set to an integer value for record TTL in seconds (if not set the default is 120)
- `"ip_version"` can be `ipv4` (A records), or `ipv6` (AAAA records) or `ipv4 or ipv6` (update one of the two, depending on the public ip found). It defaults to `ipv4 or ipv6`.
- `"ipv6_suffix"` is the IPv6 interface identifier suffix to use. It can be for example `0:0:0:0:72ad:8fbb:a54e:bedd/64`. If left empty, it defaults to no suffix and the raw public IPv6 address obtained is used in the record updating.

## Domain setup

See [their article](https://cp.servercow.de/en/plugin/support_manager/knowledgebase/view/34/dns-api-v1/7/)
