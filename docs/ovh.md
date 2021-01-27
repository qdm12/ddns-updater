# OVH

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "ovh",
      "domain": "domain.com",
      "host": "@",
      "username": "username",
      "password": "password",
      "ip_version": "ipv4",
      "provider_ip": true
    }
  ]
}
```

### Compulsory parameters

- `"domain"`
- `"host"` is your host and can be a subdomain or `"@"`

#### Using DynHost

- `"username"`
- `"password"`

#### OR Using ZoneDNS

- `"api_endpoint"` default is `"ovh-eu"`
- `"app_key"`
- `"app_secret"`
- `"consumer_key"`

The ZoneDNS implementation allows you to update any record name including *.yourdomain.tld

### Optional parameters

- `"ip_version"` can be `ipv4` (A records) or `ipv6` (AAAA records), defaults to `ipv4 or ipv6`
- `"provider_ip"` can be set to `true` to let your DNS provider determine your IPv4 address (and/or IPv6 address) automatically when you send an update request, without sending the new IP address detected by the program in the request.
- `"zone_dns"` select between two modes, OVH's dynamic hosting service (`false`) or OVH's API (`true`). Default is `false`

## Domain setup

If you are using DynHost:

https://docs.ovh.com/ie/en/domains/hosting_dynhost/

If you are using the ZoneDNS api:

https://docs.ovh.com/gb/en/customer/first-steps-with-ovh-api/