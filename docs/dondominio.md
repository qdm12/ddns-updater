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
      "password": "password",
      "ip_version": "ipv4"
    }
  ]
}
```

### Compulsory parameters

- `"domain"`
- `"name"` is the name server associated with the domain
- `"username"`
- `"password"`

### Optional parameters

- `"ip_version"` can be `ipv4` (A records) or `ipv6` (AAAA records), defaults to `ipv4 or ipv6`

## Domain setup
