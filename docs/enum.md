# enum

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "enum",
      "domain": "domain.com",
      "api_key": "enum_sk_...",
      "project_id": "proj-...",
      "ip_version": "ipv4",
      "ipv6_suffix": ""
    }
  ]
}
```

### Compulsory parameters

- `"domain"` is the domain to update. It can be `example.com` (root domain), `sub.example.com` (subdomain of `example.com`) or `*.example.com` for the wildcard.
- `"api_key"` is the API key of an enum service account.
- `"project_id"` is the ID of the enum project containing the DNS zone, for example `proj-01kmyy3t719crcnrrvk1mgyjd0`.

### Optional parameters

- `"ttl"` is the record TTL in seconds, between `60` and `86400`. It defaults to the existing TTL of the record, or to `300` for new records.
- `"ip_version"` can be `ipv4` (A records), or `ipv6` (AAAA records) or `ipv4 or ipv6` (update one of the two, depending on the public ip found). It defaults to `ipv4 or ipv6`.
- `"ipv6_suffix"` is the IPv6 interface identifier suffix to use. It can be for example `0:0:0:0:72ad:8fbb:a54e:bedd/64`. If left empty, it defaults to no suffix and the raw temporary IPv6 address of the machine is used in the record updating. You might want to set this to use your permanent IPv6 address instead of your temporary IPv6 address.

## Domain setup

1. Add your domain as a DNS zone in your enum project, for example with `enumctl dns zones create domain.com`, and point its nameservers to enum.
2. Find your project ID with `enumctl projects list`.
3. Create a service account with an API key with `enumctl service-accounts create ddns-updater --key ddns-updater`. API keys expire after 90 days by default, use `--expires-in 1y` to change this.

More information at the [enum DNS documentation](https://docs.enum.co/services/dns/).
