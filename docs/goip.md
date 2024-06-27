# GoIP.de

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "goip",
      "domain": "mysubdomain.goip.de",
      "username": "username",
      "password": "password",
      "provider_ip": true,
      "ip_version": "",
      "ipv6_suffix": ""

    }
  ]
}
```

### Compulsory parameters

- `"domain"` is the domain to update. For example, for the owner/host `sub`, it would be `sub.goip.de`. The [eTLD+1](https://developer.mozilla.org/en-US/docs/Glossary/eTLD) must be `goip.de` or `goip.it`.
- `"username"` is your goip.de username listed under "Routers"
- `"password"` is your router account password

### Optional parameters

- `"domain"` is the domain name which can be `goip.de` or `goip.it`, and defaults to `goip.de` if left unset.
- `"provider_ip"` can be set to `true` to let your DNS provider determine your IPv4 address (and/or IPv6 address) automatically when you send an update request, without sending the new IP address detected by the program in the request. This is automatically disabled for an IPv6 public address since it is not supported.
- `"ip_version"` can be `ipv4` (A records), or `ipv6` (AAAA records) or `ipv4 or ipv6` (update one of the two, depending on the public ip found). It defaults to `ipv4`.
- `"ipv6_suffix"` is the IPv6 interface identifier suffix to use. It can be for example `0:0:0:0:72ad:8fbb:a54e:bedd/64`. If left empty, it defaults to no suffix and the raw public IPv6 address obtained is used in the record updating.
