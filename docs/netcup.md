# Netcup

## Configuration

Note: This implementation does not require a domain reseller account. The warning in the dashboard can be ignored.

### Example

```json
{
  "settings": [
    {
      "provider": "netcup",
      "domain": "domain.com",
      "host": "host",
      "api_key": "xxxxx",
      "password": "api-key-password",
      "customer_number": "111111"
    }
  ]
}
```

### Compulsory parameters

- `"domain"` is your domain
- `"host"` is your host (subdomain)
- `"api_key"` is your api key (generated in the customercontrolpanel)
- `"password"` is your api password (generated in the customercontrolpanel). Netcup only allows one ApiPassword. This is not the account password.
- `"customer_number"` is your customer number (viewable in the customercontrolpanel next to your name)
