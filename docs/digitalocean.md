# Digital Ocean

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "digitalocean",
      "domain": "domain.com",
      "host": "@",
      "token": "yourtoken",
      "ip_version": "ipv4"
    }
  ]
}
```

### Compulsory parameters

- `"domain"`
- `"host"` is your host and can be a subdomain or `"@"` or `"*"`
- `"token"` is your token that you can create [here](https://cloud.digitalocean.com/settings/applications)

### Optional parameters

- `"ip_version"` can be `ipv4` (A records), or `ipv6` (AAAA records) or `ipv4 or ipv6` (update one of the two, depending on the public ip found). It defaults to `ipv4 or ipv6`.

## Domain setup
