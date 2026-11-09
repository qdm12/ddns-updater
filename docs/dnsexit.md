# DNSExit

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "dnsexit",
      "domain": "myname.linkpc.net",
      "api_key": "your-dns-api-key",
      "ip_version": "ipv4",
      "ipv6_suffix": ""
    }
  ]
}
```

### Compulsory parameters

- `"domain"` is the hostname to update, for example `myname.linkpc.net` for a free dynamic hostname or `sub.example.com` for a record of a domain hosted at DNSExit.
- `"api_key"` is the DNS API key. Create it in your DNSExit account under **Settings** then **DNS API Key**.

### Optional parameters

- `"ip_version"` can be `ipv4` (A records), or `ipv6` (AAAA records) or `ipv4 or ipv6` (update one of the two, depending on the public ip found). It defaults to `ipv4 or ipv6`.
- `"ipv6_suffix"` is the IPv6 interface identifier suffix to use. It can be for example `0:0:0:0:72ad:8fbb:a54e:bedd/64`. If left empty, it defaults to no suffix and the raw temporary IPv6 address of the machine is used in the record updating. You might want to set this to use your permanent IPv6 address instead of your temporary IPv6 address.

## Domain setup

Add the hostname you want to keep updated in the DNSExit dashboard, then create a DNS API key under **Settings** > **DNS API Key**.

The updater uses the [DNSExit dynamic IP update endpoint](https://dnsexit.com/dns/dns-api/#dynamic-ip-update). Note that DNSExit limits how often a hostname can be updated (5 updates per 10 minutes at the time of writing), so keep the update period above 4 minutes.
