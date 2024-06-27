# Porkbun

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "porkbun",
      "domain": "domain.com",
      "api_key": "sk1_7d119e3f656b00ae042980302e1425a04163c476efec1833q3cb0w54fc6f5022",
      "secret_api_key": "pk1_5299b57125c8f3cdf347d2fe0e713311ee3a1e11f11a14942b26472593e35368",
      "ip_version": "ipv4",
      "ipv6_suffix": ""
    }
  ]
}
```

### Compulsory Parameters

- `"domain"` is the domain to update. It can be `example.com` (root domain), `sub.example.com` (subdomain of `example.com`) or `*.example.com` for the wildcard.
- `"apikey"`
- `"secretapikey"`
- `"ttl"` optional integer value corresponding to a number of seconds

### Optional parameters

- `"ip_version"` can be `ipv4` (A records), or `ipv6` (AAAA records) or `ipv4 or ipv6` (update one of the two, depending on the public ip found). It defaults to `ipv4 or ipv6`.
- `"ipv6_suffix"` is the IPv6 interface identifier suffix to use. It can be for example `0:0:0:0:72ad:8fbb:a54e:bedd/64`. If left empty, it defaults to no suffix and the raw public IPv6 address obtained is used in the record updating.

## Domain setup

- Create an API key at [porkbun.com/account/api](https://porkbun.com/account/api)
- From the [Domain Management page](https://porkbun.com/account/domainsSpeedy), toggle on **API ACCESS** for your domain.

💁 [Official setup documentation](https://kb.porkbun.com/article/190-getting-started-with-the-porkbun-dns-api)

## Record creation

In case you don't have an A or AAAA record for your host and domain combination, it will be created by DDNS-Updater.
However, to do so, the corresponding ALIAS record, that is automatically created by Porkbun, is automatically deleted to allow this.
More details is in [this comment by @everydaycombat](https://github.com/qdm12/ddns-updater/issues/546#issuecomment-1773960193).
