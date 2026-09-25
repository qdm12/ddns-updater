# FENO

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "feno",
      "domain": "example.no",
      "owner": "home",
      "api_key": "feno_live_YOUR_API_KEY",
      "ip_version": "ipv4",
      "ipv6_suffix": ""
    }
  ]
}
```

### Compulsory parameters

- `"domain"` is the domain to update. It can be `example.no` (root domain) or `sub.example.no` (subdomain of `example.no`). Wildcards (`*.example.no`) are not supported.
- `"api_key"` is a FENO API key (`feno_live_...`) with the `ddns:write` scope (a key with `dns:write` also works).

### Optional parameters

- `"owner"` is the sub domain to use. It can be `@` for the root domain (default), or a name such as `home` or `nas.home` for any depth of subdomain.
- `"ip_version"` can be `ipv4` (A records), or `ipv6` (AAAA records) or `ipv4 or ipv6` (update one of the two, depending on the public ip found). It defaults to `ipv4 or ipv6`.
- `"ipv6_suffix"` is the IPv6 interface identifier suffix to use. It can be for example `0:0:0:0:72ad:8fbb:a54e:bedd/64`. If left empty, it defaults to no suffix and the raw temporary IPv6 address of the machine is used in the record updating. You might want to set this to use your permanent IPv6 address instead of your temporary IPv6 address.

## Domain setup

1. Make sure your domain uses the FENO nameservers (`ns1.feno.no` and `ns2.feno.no`); the update endpoint only works for zones hosted there.
1. In the [FENO dashboard](https://feno.no/), go to **Account → API keys** and create an API key with the `ddns:write` scope (MFA is required). Do not set an IP allowlist on the key, since your address is expected to change.
1. Set the key as `"api_key"`. The `A` and/or `AAAA` record is created automatically on the first update if it does not already exist.

Updates are rate limited to 10 per minute per key. More information is available in the [FENO dynamic DNS documentation](https://github.com/mrerikcodes/feno-api/blob/main/docs/DDNS.md).
