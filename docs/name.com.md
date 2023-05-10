# Name.com

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "name.com",
      "domain": "domain.com",
      "host": "@",
      "username": "username",
      "password": "password",
      "ttl": 300
    }
  ]
}
```

### Compulsory parameters

- `"domain"`
- `"host"` is your host and can be a subdomain, `"@"` or `"*"` generally
- `"username"`
- `"password"`

### Optional parameters

- `"ttl"` is the time this record can be cached for in seconds. Name.com allows a minimum TTL of 300, or 5 minutes. Name.com defaults to 300 if not provided.
- `"ip_version"` can be `ipv4` (A records) or `ipv6` (AAAA records), defaults to `ipv4 or ipv6`

## Domain setup

<a href="https://www.name.com"><img src="../readme/name.svg" alt="drawing" width="25%"/></a>

1. Create a Name.com account and buy a domain name - *example.com* as an example
1. Login to Name.com at [https://www.name.com/account/login](https://www.name.com/account/login)
1. Search for domains to purchase [https://www.name.com/domain/search](https://www.name.com/domain/search)

## API Credentials

1. Get your API token at [https://www.name.com/account/settings/api](https://www.name.com/account/settings/api)
1. Your username & password for the configuration will be your account username & token you generate from the API settings page.
