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
      "username": "user",
      "password": "password",
      "dual_stack": false,
      "ip_version": "ipv4",
      "ipv6_suffix": ""
    }
  ]
}
```

### Compulsory parameters

- `"domain"` is the domain to update. It can be `example.com` (root domain) or `sub.example.com` (subdomain of `example.com`).
- `"username"`
- `"password"`

### Optional parameters

- `"dual_stack"` can be set to `true` **if you have turn on dual stack for your record** to update both IPv4 and IPv6 addresses. Note it is ignored if `"provider_ip": true`. More precisely:
  - if it is `false`, the updates are done using the `ip` parameter and only one IP address can be set (ipv4 or ipv6, whichever is last sent).
  - if it is `true`, the updates are done using the `ip` and `ip6` parameters, for IPv4 and IPv6 respectively, and both can be set on the same record
- `"ip_version"` can be `ipv4` (A records), or `ipv6` (AAAA records) or `ipv4 or ipv6` (update one of the two, depending on the public ip found). It defaults to `ipv4 or ipv6`.
- `"ipv6_suffix"` is the IPv6 interface identifier suffix to use. It can be for example `0:0:0:0:72ad:8fbb:a54e:bedd/64`. If left empty, it defaults to no suffix and the raw public IPv6 address obtained is used in the record updating.

## Domain setup
