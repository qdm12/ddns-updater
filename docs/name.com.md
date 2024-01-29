# Name.com

<img src="../readme/name.svg" alt="drawing" width="25%"/>

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "name.com",
      "domain": "domain.com",
      "host": "@",
      "username": "username",
      "token": "token"
    }
  ]
}
```

### Compulsory parameters

- `"domain"`
- `"host"` is your host and can be a subdomain, `"@"` or `"*"` generally
- `"username"` is your account username
- `"token"` which you can obtain from [www.name.com/account/settings/api](https://www.name.com/account/settings/api)

### Optional parameters

- `"ttl"` is the time this record can be cached for in seconds. Name.com allows a minimum TTL of 300, or 5 minutes. Name.com defaults to 300 if not provided.
- `"ip_version"` can be `ipv4` (A records), or `ipv6` (AAAA records) or `ipv4 or ipv6` (update one of the two, depending on the public ip found). It defaults to `ipv4 or ipv6`.
