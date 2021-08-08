# Porkbun

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "porkbun",
      "domain": "domain.com",
      "host": "@",
      "api_key": "sk1_7d119e3f656b00ae042980302e1425a04163c476efec1833q3cb0w54fc6f5022",
      "secret_api_key": "pk1_5299b57125c8f3cdf347d2fe0e713311ee3a1e11f11a14942b26472593e35368",
      "ip_version": "ipv4"
    }
  ]
}
```

### Parameters

- `"domain"`
- `"host"` is your host and can be a subdomain, `"*"` or `"@"`
- `"apikey"`
- `"secretapikey"`
- `"ttl"` optional integer value corresponding to a number of seconds

## Domain setup

- Create an API key at [porkbun.com/account/api](https://porkbun.com/account/api)
- From the [Domain Management page](https://porkbun.com/account/domainsSpeedy), toggle on **API ACCESS** for your domain.

💁 [Official setup documentation](https://kb.porkbun.com/article/190-getting-started-with-the-porkbun-dns-api)
