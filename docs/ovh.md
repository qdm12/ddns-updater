# OVH

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "ovh",
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

#### Using DynHost

- `"username"`
- `"password"`

#### OR Using ZoneDNS

- `"api_endpoint"` default value is `"ovh-eu"`
- `"app_key"` which you can create at [eu.api.ovh.com/createApp](https://eu.api.ovh.com/createApp/)
- `"app_secret"`
- `"consumer_key"`

The ZoneDNS implementation allows you to update any record name including *.yourdomain.tld

### Optional parameters

- `"ip_version"` can be `ipv4` (A records), or `ipv6` (AAAA records) or `ipv4 or ipv6` (update one of the two, depending on the public ip found). It defaults to `ipv4 or ipv6`.
- `"ipv6_suffix"` is the IPv6 interface identifier suffix to use. It can be for example `0:0:0:0:72ad:8fbb:a54e:bedd/64`. If left empty, it defaults to no suffix and the raw public IPv6 address obtained is used in the record updating.
- `"provider_ip"` can be set to `true` to let your DNS provider determine your IPv4 address (and/or IPv6 address) automatically when you send an update request, without sending the new IP address detected by the program in the request.
- `"mode"` select between two modes, OVH's dynamic hosting service (`"dynamic"`) or OVH's API (`"api"`). Default is `"dynamic"`

## Domain setup

- If you use DynHost: [docs.ovh.com/ie/en/domains/hosting_dynhost](https://docs.ovh.com/ie/en/domains/hosting_dynhost/)
- If you use the ZoneDNS API: [docs.ovh.com/gb/en/customer/first-steps-with-ovh-api](https://docs.ovh.com/gb/en/customer/first-steps-with-ovh-api/)
