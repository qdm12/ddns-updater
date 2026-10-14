# Hetzner Cloud

This provider uses the Hetzner Cloud API `https://api.hetzner.cloud/v1/` which is different from the legacy Hetzner DNS API.

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "hetznercloud",
      "domain": "example.com",
      "token": "yourtoken",
      "ip_version": "ipv4",
      "ipv6_suffix": ""
    }
  ]
}
```

### Compulsory parameters

- `"domain"` is the domain to update. It can be `example.com` (root domain), `sub.example.com` (subdomain of `example.com`) or `*.example.com` for the wildcard.
- `"token"` is your API token configured with DNS write permissions for your DNS zone, see [the Authentication section](https://docs.hetzner.com/cloud/api/getting-started/generating-api-token)

### Optional parameters

- `"ip_version"` can be `ipv4` (A records), or `ipv6` (AAAA records) or `ipv4 or ipv6` (update one of the two, depending on the public ip found). It defaults to `ipv4 or ipv6`.
- `"ipv6_suffix"` is the IPv6 interface identifier suffix to use. It can be for example `0:0:0:0:72ad:8fbb:a54e:bedd/64`. If left empty, it defaults to no suffix and the raw temporary IPv6 address of the machine is used in the record updating. You might want to set this to use your permanent IPv6 address instead of your temporary IPv6 address.
- `"ttl"` time to live for the DNS record in seconds. It is only used to add a record to the rrset, and is not used to update an existing record. If left empty, it defaults to the existing zone TTL.

## RRSet ownership

Hetzner Cloud groups records with the same name and type into an RRSet. When
ddns-updater changes an existing A or AAAA RRSet, Hetzner's `set_records` action
replaces all its values with the single dynamic IP address.

Use a dedicated hostname for DDNS, for example:

```text
home.example.com A <dynamic public IPv4>
```

Other services can point to that hostname with CNAME records. Do not add
round-robin or static addresses to the managed A/AAAA RRSet if they must be
preserved.

If the desired dynamic IP is already one of several values in an RRSet,
ddns-updater currently considers the record up to date and leaves every value
unchanged. If the desired IP is absent, the subsequent update replaces the
entire RRSet as described above.

For least privilege, use a dedicated Hetzner project/API token for DNS
infrastructure instead of reusing a broader infrastructure credential.
