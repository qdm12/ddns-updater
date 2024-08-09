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
      "host": "@ or *",
      "ttl": 600,
      "token": "yourtoken",
      "ip_version": "ipv4",
      "ipv6_suffix": ""
    }
  ]
}
```

### Compulsory parameters

- `"zone_identifier"` is the Zone ID of your site, from the domain overview page written as *Zone ID*
- `"domain"` is the domain to update. It can be `example.com` (root domain) or `sub.example.com` (subdomain of `example.com`).
- `"host"` defines whether the domain root or the subdomain wildcard is udpated. `"@"` defines root `"domain.com"` and `"*"` defines all subdomains such as `"sub.domain.com"` OR `"test.sub.domain.com"`. Inaddition the wildcard can be used to define subdomains such as `"sub.*"` or other permutations as you desire which represents the example `"sub.test.domain.com"`.
- `"ttl"` integer value for record TTL in seconds (specify 1 for automatic)
- One of the following ([how to find API keys](https://developers.cloudflare.com/fundamentals/api/get-started/)):
  - Email `"email"` and Global API Key `"key"`
  - User service key `"user_service_key"`
  - API Token `"token"`, configured with DNS edit permissions for your DNS name's zone

### Optional parameters

- `"proxied"` can be set to `true` to use the proxy services of Cloudflare
- `"ip_version"` can be `ipv4` (A records), or `ipv6` (AAAA records) or `ipv4 or ipv6` (update one of the two, depending on the public ip found). It defaults to `ipv4 or ipv6`.
- `"ipv6_suffix"` is the IPv6 interface identifier suffix to use. It can be for example `0:0:0:0:72ad:8fbb:a54e:bedd/64`. If left empty, it defaults to no suffix and the raw public IPv6 address obtained is used in the record updating.

Special thanks to @Starttoaster for helping out with the [documentation](https://gist.github.com/Starttoaster/07d568c2a99ad7631dd776688c988326) and testing.
