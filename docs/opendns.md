# OpenDNS

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "opendns",
      "domain": "example.com",
      "hostname": "Home",
      "username": "username",
      "password": "password",
      "ip_version": "ipv4",
      "ipv6_suffix": ""
    }
  ]
}
```

### Compulsory parameters

- `"domain"` is the domain name of the record, for example `example.com` (root domain) or `sub.example.com` (subdomain of `example.com`). It identifies the record in the logs, the HTTP API and the database, and it is used as the name of the network to update unless the optional `"hostname"` parameter is set.
- `"username"`
- `"password"`

### Optional parameters

- `"hostname"` is the name of the network to update, as declared in your OpenDNS account, for example `Home`. It is used as it is, and defaults to the value of `"domain"`. Only one network is updated per settings entry, use one entry per network.
- `"ip_version"` can be `ipv4` (A records), or `ipv6` (AAAA records) or `ipv4 or ipv6` (update one of the two, depending on the public ip found). It defaults to `ipv4 or ipv6`.
- `"ipv6_suffix"` is the IPv6 interface identifier suffix to use. It can be for example `0:0:0:0:72ad:8fbb:a54e:bedd/64`. If left empty, it defaults to no suffix and the raw temporary IPv6 address of the machine is used in the record updating. You might want to set this to use your permanent IPv6 address instead of your temporary IPv6 address.

## Network setup

The network name must match a network declared in your OpenDNS account, under *Settings > Network*, with *Dynamic IP Update* enabled.

Because OpenDNS updates the IP address of that network and not the IP address of a DNS record, an update is sent when the public IP address differs from the last one sent, like for proxied Cloudflare records. The `"domain"` value is then only an identifier and does not need to be published in DNS.
