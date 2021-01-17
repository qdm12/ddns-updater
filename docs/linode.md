# Linode

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "linode",
      "domain": "domain.com",
      "host": "@",
      "token": "token",
      "ip_version": "ipv4"
    }
  ]
}
```

### Compulsory parameters

- `"domain"`
- `"host"` is your host and can be a subdomain or `"@"` or `"*"`
- `"token"`

### Optional parameters

- `"ip_version"` can be `ipv4` (A records) or `ipv6` (AAAA records), defaults to `ipv4 or ipv6`

## Domain setup

1. Create a personal access token with the [Linode's guide](https://www.linode.com/docs/products/tools/cloud-manager/guides/cloud-api-keys) and use it as `token`.
1. The program will create the A or AAAA record for you if it doesn't exist already.
