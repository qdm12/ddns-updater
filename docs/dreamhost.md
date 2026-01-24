# Dreamhost

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "dreamhost",
      "domain": "domain.com",
      "key": "key",
      "ip_version": "ipv4",
      "ipv6_suffix": ""
    }
  ]
}
```

### Compulsory parameters

- `"domain"`
- `"key"`

### Optional parameters

- `"domain"` is the domain to update. It can be `example.com` (root domain) or `sub.example.com` (subdomain of `example.com`).
- `"ip_version"` can be `ipv4` (A records), or `ipv6` (AAAA records) or `ipv4 or ipv6` (update one of the two, depending on the public ip found). It defaults to `ipv4 or ipv6`.
- `"ipv6_suffix"` is the IPv6 interface identifier suffix to use. It can be for example `0:0:0:0:72ad:8fbb:a54e:bedd/64`. If left empty, it defaults to no suffix and the raw public IPv6 address obtained is used in the record updating.

## Domain setup

1. Login to the Dreamhost control panel and navigate to the API key page. https://panel.dreamhost.com/?tree=home.api
1. Generate a new API Key with a comment to describe its purpose. Add permissions("Functions this key should have access to:") for **All dns functions**.
1. Add your key to your configuration settings for ddns-updater (config.json).
1. Confirm your domain's DNS records already as an **A** name custom record established in the Dreamhost control panel. If no A name is listed, ddns-updater will fail with error "no_such_zone."
