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
      "host": "host",
      "api_key": "xxxxx",
      "password": "yyyyy",
      "customer_number": "111111"
    }
  ]
}
```

### Compulsory parameters

- `"domain"` is your domain
- `"host"` is your host (subdomain) or `"@"` for the root of the domain. It cannot be the wildcard.
- `"api_key"` is your api key (generated in the [customercontrolpanel](https://www.customercontrolpanel.de))
- `"password"` is your api password (generated in the [customercontrolpanel](https://www.customercontrolpanel.de)). Netcup only allows one ApiPassword. This is not the account password. This password is used for all api keys.
- `"customer_number"` is your customer number (viewable in the [customercontrolpanel](https://www.customercontrolpanel.de) next to your name). As seen in the example above, provide the number as string value.
