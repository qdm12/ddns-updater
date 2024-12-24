# [Myaddr](https://myaddr.tools/)

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "myaddr",
      "domain": "your-name.myaddr.tools",
      "key": "key",
      "ip_version": "ipv4",
      "ipv6_suffix": ""
    }
  ]
}
```

### Compulsory parameters

- `"domain"` - the **single** domain to update; note the `key` below updates all records and subdomains for this domain. It should be `your-name*.myaddr.tools`.
- `"key"` - the private key corresponding to the domain to update

### Optional parameters

- `"ip_version"` can be `ipv4` (A records), or `ipv6` (AAAA records) or `ipv4 or ipv6` (update one of the two, depending on the public ip found). It defaults to `ipv4 or ipv6`.
- `"ipv6_suffix"` is the IPv6 interface identifier suffix to use. It can be for example `0:0:0:0:72ad:8fbb:a54e:bedd/64`. If left empty, it defaults to no suffix and the raw public IPv6 address obtained is used in the record updating.

## Domain setup

Claim a subdomain at [myaddr.tools](https://myaddr.tools/)
