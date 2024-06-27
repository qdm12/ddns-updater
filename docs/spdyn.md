# Spdyn.de

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "spdyn",
      "domain": "domain.com",
      "user": "user",
      "password": "password",
      "token": "token",
      "ip_version": "ipv4",
      "ipv6_suffix": "",
      "provider_ip": true
    }
  ]
}
```

### Compulsory parameters

- `"domain"` is the domain to update. It can be `example.com` (root domain) or `sub.example.com` (subdomain of `example.com`).

#### Using user and password

- `"user"` is the name of a user who can update this host
- `"password"` is the password of a user who can update this host

#### Using update tokens

- `"token"` is your update token

### Optional parameters

- `"ip_version"` can be `ipv4` (A records), or `ipv6` (AAAA records) or `ipv4 or ipv6` (update one of the two, depending on the public ip found). It defaults to `ipv4 or ipv6`.
- `"ipv6_suffix"` is the IPv6 interface identifier suffix to use. It can be for example `0:0:0:0:72ad:8fbb:a54e:bedd/64`. If left empty, it defaults to no suffix and the raw public IPv6 address obtained is used in the record updating.
- `"provider_ip"` can be set to `true` to let your DNS provider determine your IPv4 address (**not IPv6**)automatically when you send an update request, without sending the new IP address detected by the program in the request.
