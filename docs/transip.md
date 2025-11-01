# TransIP

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "transip",
      "domain": "example.com",
      "ip_version":  "ipv4",
      "ipv6_suffix": "",
      "username": "username",
      "key": "-----BEGIN PRIVATE KEY-----...-----END PRIVATE KEY-----"
    }
  ]
}
```

### Compulsory parameters

- `"domain"` is the domain to update. It can be `example.com` (root domain), `sub.example.com` (subdomain of `example.com`) or `*.example.com` for the wildcard.
- `"username"`
- `"key"`

### Optional parameters

- `"ip_version"` can be `ipv4` (A records), or `ipv6` (AAAA records) or `ipv4 or ipv6` (update one of the two, depending on the public ip found). It defaults to `ipv4 or ipv6`.
- `"ipv6_suffix"` is the IPv6 interface identifier suffix to use. It can be for example `0:0:0:0:72ad:8fbb:a54e:bedd/64`. If left empty, it defaults to no suffix and the raw public IPv6 address obtained is used in the record updating.

## Domain setup

1. Log in to the [TransIP control panel](https://www.transip.nl/cp/).
2. Under your account, go to the [API settings](https://www.transip.nl/cp/account/api/).
3. Enable the API.\
![A toggle showing the API status as on](../readme/transip1.png)
4. Add a key pair. Make sure to uncheck the checkbox to only accept IP addresses from the whitelist.\
![A table listing the key pairs](../readme/transip2.png)
5. Copy your private key, and store it in your config file.\
![A snippet of a generated private key](../readme/transip3.png)
