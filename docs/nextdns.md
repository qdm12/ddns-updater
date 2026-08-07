# NextDNS

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "nextdns",
      "domain": "nextdns.io",
      "owner": "link-ip",
      "endpoint_id": "abcdef",
      "api_guid": "0123456789abcdef",
      "ip_version": "ipv4",
      "ipv6_suffix": ""
    }
  ]
}
```

### Compulsory parameters

- `"domain"` must be `"nextdns.io"` and `"owner"` must be `"link-ip"`.
- `"endpoint_id"` is the first path segment of the Linked IP update URL.
- `"api_guid"` is the second path segment of the Linked IP update URL.

### Optional parameters

- `"ip_version"` can be `ipv4` (A records), or `ipv6` (AAAA records) or `ipv4 or ipv6` (update one of the two, depending on the public ip found). It defaults to `ipv4 or ipv6`.
- `"ipv6_suffix"` is the IPv6 interface identifier suffix to use. It can be for example `0:0:0:0:72ad:8fbb:a54e:bedd/64`. If left empty, it defaults to no suffix and the raw temporary IPv6 address of the machine is used in the record updating. You might want to set this to use your permanent IPv6 address instead of your temporary IPv6 address.

## Domain setup

NextDNS supports updating Linked IP via a DDNS hostname. If you're already using a DDNS service, configure your DDNS domain in the Linked IP card instead.

1. Create an account on the [nextdns website](https://nextdns.io/)
1. Go to your [account page](https://my.nextdns.io/), login and setup Linked IP
1. Click `Show advanced options` in the Linked IP card
1. Copy the update URL shown — it looks like `https://link-ip.nextdns.io/abcdef/0123456789abcdef`
1. Set `"endpoint_id"` to the first path segment (e.g. `abcdef`) and `"api_guid"` to the second path segment (e.g. `0123456789abcdef`)

See the [nextdns website](https://nextdns.io/)
