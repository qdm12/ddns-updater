# Cloudflare

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "cloudflare",
      "zone_identifier": "some id",
      "domain": "domain.com",
      "host": "@",
      "ttl": 600,
      "token": "yourtoken",
      "ip_version": "ipv4"
    }
  ]
}
```

### Compulsory parameters

- `"zone_identifier"` is the Zone ID of your site, from the domain overview page written as *Zone ID*
- `"domain"`
- `"host"` is your host and can be `"@"`, a subdomain or the wildcard `"*"`.
See [this issue comment for context](https://github.com/qdm12/ddns-updater/issues/243#issuecomment-928313949). This is left as is for compatibility.
- `"ttl"` integer value for record TTL in seconds (specify 1 for automatic)
- One of the following ([how to find API keys](https://support.cloudflare.com/hc/en-us/articles/200167836-Where-do-I-find-my-Cloudflare-API-key-)):
  - Email `"email"` and Global API Key `"key"`
  - User service key `"user_service_key"`
  - API Token `"token"`, configured with DNS edit permissions for your DNS name's zone

### Optional parameters

- `"proxied"` can be set to `true` to use the proxy services of Cloudflare
- `"ip_version"` can be `ipv4` (A records) or `ipv6` (AAAA records), and defaults to `ipv4 or ipv6`

Special thanks to @Starttoaster for helping out with the [documentation](https://gist.github.com/Starttoaster/07d568c2a99ad7631dd776688c988326) and testing.
