# Gigahost.no

## Configuration

### Example with an API key

```json
{
  "settings": [
    {
      "provider": "gigahostno",
      "domain": "sub.example.no",
      "apikey": "flux_live_YOUR_API_KEY",
      "ip_version": "ipv4",
      "ipv6_suffix": ""
    }
  ]
}
```

### Example with email and password

```json
{
  "settings": [
    {
      "provider": "gigahostno",
      "domain": "sub.example.no",
      "email": "you@example.no",
      "password": "your password",
      "ip_version": "ipv4",
      "ipv6_suffix": ""
    }
  ]
}
```

### Compulsory parameters

- `"domain"` is the domain to update. It can be `example.no` (root domain), `sub.example.no` (subdomain of `example.no`) or `*.example.no` for the wildcard.
- Authentication, using **either**:
  - `"apikey"` is an API key generated from your Gigahost account. This is the recommended method.
  - `"email"` (your account email) together with `"password"` (your account password).

> ⚠️ Two-factor authentication (2FA) is not supported for now when using `"email"` and `"password"`. If you have 2FA enabled on your account, use an `"apikey"` instead.

### Optional parameters

- `"ip_version"` can be `ipv4` (A records), or `ipv6` (AAAA records) or `ipv4 or ipv6` (update one of the two, depending on the public ip found). It defaults to `ipv4 or ipv6`.
- `"ipv6_suffix"` is the IPv6 interface identifier suffix to use. It can be for example `0:0:0:0:72ad:8fbb:a54e:bedd/64`. If left empty, it defaults to no suffix and the raw temporary IPv6 address of the machine is used in the record updating. You might want to set this to use your permanent IPv6 address instead of your temporary IPv6 address.

## Domain setup

1. Log in to your [Gigahost account](https://gigahost.no/) and make sure your domain's DNS is hosted at Gigahost.
2. To use an API key, generate one from your account and set it as `"apikey"`. Otherwise set your account email as `"email"` and your password as `"password"`.

The record (`A` for IPv4 and/or `AAAA` for IPv6) is created automatically on the first update if it does not already exist.

More information is available in the [Gigahost API documentation](https://gigahost.no/en/api-dokumentasjon).
