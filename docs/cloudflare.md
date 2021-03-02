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

- `"zone_identifier"` is the Zone ID of your site
- `"domain"`
- `"host"` is your host and can be a subdomain, `"@"` or `"*"` generally
- `"ttl"` integer value for record TTL in seconds (specify 1 for automatic)
- One of the following:
    - Email `"email"` and Global API Key `"key"`
    - User service key `"user_service_key"`
    - API Token `"token"`, configured with DNS edit permissions for your DNS name's zone

### Optional parameters

- `"proxied"` can be set to `true` to use the proxy services of Cloudflare
- `"ip_version"` can be `ipv4` (A records) or `ipv6` (AAAA records), and defaults to `ipv4 or ipv6`

## Domain setup

1. Make sure you have `curl` installed
1. Obtain your API key from Cloudflare website ([see this](https://support.cloudflare.com/hc/en-us/articles/200167836-Where-do-I-find-my-Cloudflare-API-key-))
1. Obtain your zone identifier for your domain name, from the domain's overview page written as *Zone ID*
1. Find your **identifier** in the `id` field with

    ```sh
    ZONEID=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
    EMAIL=example@example.com
    APIKEY=aaaaaaaaaaaaaaaaaa
    curl -X GET "https://api.cloudflare.com/client/v4/zones/$ZONEID/dns_records" \
        -H "X-Auth-Email: $EMAIL" \
        -H "X-Auth-Key: $APIKEY"
    ```

You can now fill in the necessary parameters in *config.json*

Special thanks to @Starttoaster for helping out with the [documentation](https://gist.github.com/Starttoaster/07d568c2a99ad7631dd776688c988326) and testing.
