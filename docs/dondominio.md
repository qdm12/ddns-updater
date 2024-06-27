# Don Dominio

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "dondominio",
      "domain": "domain.com",
      "name": "something",
      "username": "username",
      "key": "key",
      "ip_version": "ipv4",
      "ipv6_suffix": ""
    }
  ]
}
```

### Compulsory parameters

- `"domain"` is the domain to update. It can be `example.com` (root domain), `sub.example.com` (subdomain of `example.com`) or `*.example.com` for the wildcard.
- `"name"` is the name of the service/hosting
- `"username"`
- `"password"`

### Optional parameters

- `"ip_version"` can be `ipv4` (A records), or `ipv6` (AAAA records) or `ipv4 or ipv6` (update one of the two, depending on the public ip found). It defaults to `ipv4 or ipv6`.
- `"ipv6_suffix"` is the IPv6 interface identifier suffix to use. It can be for example `0:0:0:0:72ad:8fbb:a54e:bedd/64`. If left empty, it defaults to no suffix and the raw public IPv6 address obtained is used in the record updating.

## Domain setup

See [dondominio.dev/en/dondns/docs/api/#before-start](https://dondominio.dev/en/dondns/docs/api/#before-start)
