# Example.com

## Configuration

If something is unclear in the documentation below, please refer to the [scaleway API documentation](https://www.scaleway.com/en/developers/api/domains-and-dns/#path-records-update-records-within-a-dns-zone).

### Example

```json
{
    "settings": [
        {
            "provider": "scaleway",
            "domain": "munchkin-academia.eu",   // corresponds to the `dns-zone` in the API documentation
            "secret_key": "<SECRET_KEY>",
            "ip_version": "ipv4",
            "ipv6_suffix": "",
            "field_type": "A",     // optional, it will default to "A"
            "field_name": "www",   // optional, it will default to "" (equivalent to "@")
            "ttl": 450             // optional, it will default to 3600
        
        }
    ]
}
```

### Compulsory parameters

- `"domain"` is the domain to update. It can be `example.com` (root domain), `sub.example.com` (subdomain of `example.com`) or `*.example.com` for the wildcard. This fields corresponds to the `dns-zone` in the scaleway API documentation.
- `"secret_key"`

### Optional parameters

- `"ip_version"` can be `"ipv4"` or `"ipv6"`. It defaults to `"ipv4"`.
- `"ipv6_suffix"` is the suffix to append to the IPv6 address. It defaults to `""`.
- `"field_type"` is the type of DNS record to update. It can be `"A"` or `"AAAA"`. It defaults to `"A"`.
- `"field_name"` is the name of the DNS record to update. For example, it could be `"www"`, `"@"` or `"*"` for the wildcard. It defaults to to `""` (equivalent to `"@"`).
- `"ttl"` is the TTL of the DNS record to update. It defaults to `3600`.

## Domain setup

If you need more information about how to configure your domain, you can check the [scaleway official documentation](https://www.scaleway.com/en/docs/network/domains-and-dns/).
