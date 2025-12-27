# Vercel

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "vercel",
      "domain": "domain.com",
      "token": "yourtoken",
      "ip_version": "ipv4",
      "ipv6_suffix": ""
    }
  ]
}
```

### Compulsory parameters

- `"domain"` is the domain to update. It can be `example.com` (root domain), `sub.example.com` (subdomain of `example.com`) or `*.example.com` for the wildcard.
- `"token"` is your Vercel API token. You can create one at [Vercel Account Settings > Tokens](https://vercel.com/account/tokens).

### Optional parameters

- `"team_id"` is your Vercel team ID. Required if the domain belongs to a team rather than your personal account. You can find this in your team settings.
- `"ttl"` is the TTL in seconds for the DNS record. Defaults to `60` if not specified.
- `"ip_version"` can be `ipv4` (A records), or `ipv6` (AAAA records) or `ipv4 or ipv6` (update one of the two, depending on the public ip found). It defaults to `ipv4 or ipv6`.
- `"ipv6_suffix"` is the IPv6 interface identifier suffix to use. It can be for example `0:0:0:0:72ad:8fbb:a54e:bedd/64`. If left empty, it defaults to no suffix and the raw public IPv6 address obtained is used in the record updating.

## Domain setup

1. Ensure your domain is added to Vercel. You can do this via the [Vercel Dashboard](https://vercel.com/dashboard) under your project's Domain settings, or via the API.
2. Your domain must be using Vercel's nameservers or have DNS management delegated to Vercel for this provider to work.
3. Create an API token with appropriate permissions at [Vercel Account Settings > Tokens](https://vercel.com/account/tokens).
4. If the domain belongs to a team, make sure to include the `team_id` parameter in your configuration.

## Example with team

```json
{
  "settings": [
    {
      "provider": "vercel",
      "domain": "domain.com",
      "token": "yourtoken",
      "team_id": "team_xxxxxxxxxx",
      "ttl": 300,
      "ip_version": "ipv4"
    }
  ]
}
```

