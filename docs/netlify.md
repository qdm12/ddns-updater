# Netlify

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "netlify",
      "domain": "domain.com",
      "token": "your-netlify-access-token",
      "ip_version": "ipv4",
      "ipv6_suffix": ""
    }
  ]
}
```

### Compulsory parameters

- `"domain"` is the domain to update. It can be `example.com` (root domain), `sub.example.com` (subdomain of `example.com`) or `*.example.com` for wildcard.
- `"token"` is your Netlify personal access token with DNS zone permissions

### Optional parameters

- `"ip_version"` can be `ipv4` (A records), or `ipv6` (AAAA records) or `ipv4 or ipv6` (update one of the two, depending on the public IP found). It defaults to `ipv4 or ipv6`.
- `"ipv6_suffix"` is IPv6 interface identifier suffix to use. It can be for example `0:0:0:0:72ad:8fbb:a54e:bedd/64`. If left empty, it defaults to no suffix and the raw public IPv6 address obtained is used in record updating.

## Domain setup

[![Netlify Website](../readme/netlify.png)](https://www.netlify.com)

1. Login to your Netlify account at [https://app.netlify.com/](https://app.netlify.com/)

2. Navigate to **Site settings** for your site

3. Go to **Domain management** → **DNS zones**

4. Ensure your domain is properly configured as a DNS zone in Netlify

## Token setup

1. Go to **User settings** → **Applications** → **Personal access tokens**

2. Click **New access token**

3. Give the token a descriptive name (e.g., "DDNS Updater")

4. Select the following scopes:
   - **DNS:read** - Read DNS zones and records
   - **DNS:edit** - Edit DNS zones and records

5. Click **Generate token**

6. Copy the generated token - this is your `"token"` value

## Testing

1. Go to your Netlify site's DNS management page

2. Check the current DNS record for your domain

3. Run ddns-updater

4. Refresh the Netlify DNS page to verify the update occurred

## Notes

- Netlify's DNS API requires the domain to be configured as a DNS zone in your Netlify account
- The provider automatically finds the appropriate DNS zone for your domain
- If a DNS record doesn't exist, it will be created
- If a DNS record exists with a different IP, it will be deleted and recreated with the new IP
- The default TTL for created records is 3600 seconds (1 hour)
- IPv6 is supported if your Netlify account has IPv6 enabled for the DNS zone