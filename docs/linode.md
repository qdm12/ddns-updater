# Linode

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "linode",
      "domain": "domain.com",
      "token": "token",
      "ip_version": "ipv4",
      "ipv6_suffix": ""
    }
  ]
}
```

### Compulsory parameters

- `"domain"` is the domain to update. It can be `example.com` (root domain), `sub.example.com` (subdomain of `example.com`) or `*.example.com` for the wildcard.
- `"token"`

### Optional parameters

- `"ip_version"` can be `ipv4` (A records), or `ipv6` (AAAA records) or `ipv4 or ipv6` (update one of the two, depending on the public ip found). It defaults to `ipv4 or ipv6`.
- `"ipv6_suffix"` is the IPv6 interface identifier suffix to use. It can be for example `0:0:0:0:72ad:8fbb:a54e:bedd/64`. If left empty, it defaults to no suffix and the raw public IPv6 address obtained is used in the record updating.

## Domain setup

1. Create a personal access token with `domains` set, with read and write privileges, ideally that never expires. You can refer to [@AnujRNair's comment](https://github.com/qdm12/ddns-updater/pull/144#discussion_r559292678) and to [Linode's guide](https://www.linode.com/docs/products/tools/api/guides/manage-api-tokens/).
1. The program will create the A or AAAA record for you if it doesn't exist already.
