# ApertoDNS

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "apertodns",
      "domain": "domain.com",
      "token": "your_api_token",
      "ip_version": "ipv4",
      "ipv6_suffix": ""
    }
  ]
}
```

### Compulsory parameters

- `"domain"`: Your ApertoDNS domain, for example `a.domain.com` or `domain.com`
- `"token"`: Your ApertoDNS API token, created from Settings -> API Keys in your
  dashboard.

### Optional parameters

- `"ip_version"` can be `ipv4` for A records, `ipv6` for AAAA records or
  `ipv4 or ipv6` to update one of the two depending on the public IP found. It
  defaults to `ipv4 or ipv6`.
- `"ipv6_suffix"` is the IPv6 interface identifier suffix to use. It can be for
  example `0:0:0:0:72ad:8fbb:a54e:bedd/64`. If left empty, it defaults to no
  suffix and the raw temporary IPv6 address of the machine is used in the record
  updating. You might want to set this to use your permanent IPv6 address
  instead of your temporary IPv6 address.
- `"api_endpoint"` is the base URL of the ApertoDNS-compatible server to use. It
  defaults to `https://api.apertodns.com`. Set this to point to a self-hosted
  server.
