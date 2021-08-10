# Ionos

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "ionos",
      "domain": "domain.com",
      "host": "@",
      "api_key": "api_key",
      "ip_version": "ipv4"
    }
  ]
}
```

### Compulsory parameters

- `"domain"`
- `"host"` is your host and can be a subdomain or `"@"` or `"*"`
- `"api_key"` is your API key, obtained from [creating an API key](https://www.ionos.com/help/domains/configuring-your-ip-address/set-up-dynamic-dns-with-company-name/#c181598)

### Optional parameters

- `"ip_version"` can be `ipv4` (A records) or `ipv6` (AAAA records), defaults to `ipv4 or ipv6`
