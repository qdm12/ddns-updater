# NextDNS

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "nextdns",
      "domain": "link-ip.nextdns.io",
      "endpoint": "endpoint",
      "ip_version": "ipv4",
      "ipv6_suffix": ""
    }
  ]
}
```

### Compulsory parameters

- `"domain"` is the domain to update. For now, it must be "link-ip.nextdns.io".
- `"endpoint"`

### Optional parameters

- `"ip_version"` can be `ipv4` (A records), or `ipv6` (AAAA records) or `ipv4 or ipv6` (update one of the two, depending on the public ip found). It defaults to `ipv4 or ipv6`.
- `"ipv6_suffix"` is the IPv6 interface identifier suffix to use. It can be for example `0:0:0:0:72ad:8fbb:a54e:bedd/64`. If left empty, it defaults to no suffix and the raw public IPv6 address obtained is used in the record updating.

## Domain setup

> NextDNS supports updating Linked IP via a DDNS hostname. If you're already using a DDNS service, configure your DDNS domain in the Linked IP card instead.

- Create an account on the [nextdns website](https://nextdns.io/)
- Go to your [account page](https://my.nextdns.io/), login and setup Linked IP
- Click `Show advanced options` button and copy the endpoint from the Linked IP card
- Update the configuration file with the endpoint

_See the [nextdns website](https://nextdns.io/)_
