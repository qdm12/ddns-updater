# Dynu

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "dynu",
      "domain": "domain.com",
      "group": "group",
      "username": "username",
      "password": "password",
      "ip_version": "ipv4",
      "ipv6_suffix": ""
    }
  ]
}
```

### Compulsory parameters

- `"domain"` is the domain to update. It can be `example.com` (root domain) or `sub.example.com` (subdomain of `example.com`).
- `"username"`
- `"password"` could be plain text or password in MD5 or SHA256 format (There's also an option for setting a password for IP Update only)

### Optional parameters

- `"ip_version"` can be `ipv4` (A records), or `ipv6` (AAAA records) or `ipv4 or ipv6` (update one of the two, depending on the public ip found). It defaults to `ipv4 or ipv6`.
- `"ipv6_suffix"` is the IPv6 interface identifier suffix to use. It can be for example `0:0:0:0:72ad:8fbb:a54e:bedd/64`. If left empty, it defaults to no suffix and the raw public IPv6 address obtained is used in the record updating.
- `"group"` specify the Group for which you want to set the IP (will update any domains and subdomains in the same group)

## Domain setup
