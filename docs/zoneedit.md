# Zoneedit

## Configuration

⚠️ zoneedit.com for some reason requires at least a 10 minutes period between update request sent.

DDNS-Updater only sends update requests when it detects your domain name IP address mismatches your current public IP address,
so it should be fine in most cases since this happens rarely (in hours/days). But in case it happens and you want to avoid this,
set the environment variable as `PERIOD=11m` to check your public IP address and update every 11 minutes only.

### Example

```json
{
  "settings": [
    {
      "provider": "zoneedit",
      "domain": "domain.com",
      "username": "username",
      "token": "token",
      "ip_version": "ipv4",
      "ipv6_suffix": "",
      "provider_ip": true
    }
  ]
}
```

### Compulsory parameters

- `"domain"` is the domain to update. It can be `example.com` (root domain), `sub.example.com` (subdomain of `example.com`) or `*.example.com` for the wildcard.
- `"username"`
- `"token"`

### Optional parameters

- `"ip_version"` can be `ipv4` (A records), or `ipv6` (AAAA records) or `ipv4 or ipv6` (update one of the two, depending on the public ip found). It defaults to `ipv4 or ipv6`.
- `"ipv6_suffix"` is the IPv6 interface identifier suffix to use. It can be for example `0:0:0:0:72ad:8fbb:a54e:bedd/64`. If left empty, it defaults to no suffix and the raw public IPv6 address obtained is used in the record updating.
- `"provider_ip"` can be set to `true` to let your DNS provider determine your IPv4 address (and/or IPv6 address) automatically when you send an update request, without sending the new IP address detected by the program in the request.

## Domain setup

[support.zoneedit.com/en/knowledgebase/article/dynamic-dns](https://support.zoneedit.com/en/knowledgebase/article/dynamic-dns)
