# Infomaniak

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "infomaniak",
      "domain": "domain.com",
      "username": "username",
      "password": "password",
      "ip_version": "ipv4",
      "ipv6_suffix": "",
      "provider_ip": true
    }
  ]
}
```

### Compulsory parameters

- `"domain"` is the domain to update. It can be `example.com` (root domain) or `sub.example.com` (subdomain of `example.com`).
- `"username"` for dyndns (**not** your infomaniak admin username!)
- `"password"` for dyndns (**not** your infomaniak admin password!)

### Optional parameters

- `"ip_version"` can be `ipv4` (A records), or `ipv6` (AAAA records) or `ipv4 or ipv6` (update one of the two, depending on the public ip found). It defaults to `ipv4 or ipv6`.
- `"ipv6_suffix"` is the IPv6 interface identifier suffix to use. It can be for example `0:0:0:0:72ad:8fbb:a54e:bedd/64`. If left empty, it defaults to no suffix and the raw public IPv6 address obtained is used in the record updating.
- `"provider_ip"` can be set to `true` to let your DNS provider determine your IPv4 address (and/or IPv6 address) automatically when you send an update request, without sending the new IP address detected by the program in the request.

## Domain setup

Follow [this guide](https://www.infomaniak.com/en/support/faq/2357/getting-started-guide-dyndns-with-an-infomaniak-domain) to set up your subdomain including `username` and `password` for use in the configuration. **do not use your infomaniak admin username and password in the configuration!**

If you only plan on using IPv4, add your current IPv4 Address. If you only plan on using IPv6, add your current IPv6 Address. If you plan to use dual-stack (IPv4 and IPv6) addresses, it does not matter what ip-address you put in the dialog.
