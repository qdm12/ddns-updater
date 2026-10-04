# Example.com

## Configuration

If something is unclear in the documentation below, please refer to the [scaleway API documentation](https://www.scaleway.com/en/developers/api/domains-and-dns/#path-records-update-records-within-a-dns-zone).

### Example

```json
{
    "settings": [
        {
            "provider": "scaleway",
            "domain": "domain.com",
            "secret_key": "<SECRET_KEY>",
            "ip_version": "ipv4",
            "ipv6_suffix": ""
        }
    ]
}
```

### Compulsory parameters

- `"domain"` is the domain to update. It can be `example.com` (root domain), `sub.example.com` (subdomain of `example.com`) or `*.example.com` for the wildcard.
- `"secret_key"` is your secret key

### Optional parameters

- `"ip_version"` can be `ipv4` (A records), or `ipv6` (AAAA records) or `ipv4 or ipv6` (update one of the two, depending on the public ip found). It defaults to `ipv4 or ipv6`.
- `"ipv6_suffix"` is the IPv6 interface identifier suffix to use. It can be for example `0:0:0:0:72ad:8fbb:a54e:bedd/64`. If left empty, it defaults to no suffix and the raw temporary IPv6 address of the machine is used in the record updating. You might want to set this to use your permanent IPv6 address instead of your temporary IPv6 address.
- `"ttl"` is the TTL of the DNS record to update, if you want to specify it.

## Domain setup

If you need more information about how to configure your domain, you can check the [scaleway official documentation](https://www.scaleway.com/en/docs/network/domains-and-dns/).
