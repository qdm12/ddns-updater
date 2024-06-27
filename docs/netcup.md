# Netcup

## Configuration

Note: This implementation does not require a domain reseller account. The warning in the dashboard can be ignored.

Also keep in mind, that TTL, Expire, Retry and Refresh values of the given Domain are not updated. They can be manually set in the dashboard. For DDNS purposes low numbers should be used.

### Example

```json
{
  "settings": [
    {
      "provider": "netcup",
      "domain": "domain.com",
      "api_key": "xxxxx",
      "password": "yyyyy",
      "customer_number": "111111",
      "ip_version": "ipv4",
      "ipv6_suffix": ""
    }
  ]
}
```

### Compulsory parameters

- `"domain"` is the domain to update. It can be `example.com` (root domain) or `sub.example.com` (subdomain of `example.com`).
- `"api_key"` is your api key (generated in the [customercontrolpanel](https://www.customercontrolpanel.de))
- `"password"` is your api password (generated in the [customercontrolpanel](https://www.customercontrolpanel.de)). Netcup only allows one ApiPassword. This is not the account password. This password is used for all api keys.
- `"customer_number"` is your customer number (viewable in the [customercontrolpanel](https://www.customercontrolpanel.de) next to your name). As seen in the example above, provide the number as string value.

### Optional parameters

- `"ip_version"` can be `ipv4` (A records), or `ipv6` (AAAA records) or `ipv4 or ipv6` (update one of the two, depending on the public ip found). It defaults to `ipv4 or ipv6`.
- `"ipv6_suffix"` is the IPv6 interface identifier suffix to use. It can be for example `0:0:0:0:72ad:8fbb:a54e:bedd/64`. If left empty, it defaults to no suffix and the raw public IPv6 address obtained is used in the record updating.
