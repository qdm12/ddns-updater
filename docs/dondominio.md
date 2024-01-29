# Don Dominio

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "dondominio",
      "domain": "domain.com",
      "host": "@",
      "name": "something",
      "username": "username",
      "key": "key",
      "ip_version": "ipv4"
    }
  ]
}
```

### Compulsory parameters

- `"domain"`
- `"host"` is the subdomain to update which can be `@`, `*` or a subdomain
- `"name"` is the name of the service/hosting
- `"username"`
- `"password"`

### Optional parameters

- `"ip_version"` can be `ipv4` (A records), or `ipv6` (AAAA records) or `ipv4 or ipv6` (update one of the two, depending on the public ip found). It defaults to `ipv4 or ipv6`.

## Domain setup

See [dondominio.dev/en/dondns/docs/api/#before-start](https://dondominio.dev/en/dondns/docs/api/#before-start)
