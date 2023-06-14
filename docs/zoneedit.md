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
      "host": "@",
      "username": "username",
      "token": "token",
      "ip_version": "ipv4",
      "provider_ip": true
    }
  ]
}
```

### Compulsory parameters

- `"domain"`
- `"host"` is your host and can be a subdomain or `"@"` or `"*"`
- `"username"`
- `"token"`

### Optional parameters

- `"ip_version"` can be `ipv4` (A records) or `ipv6` (AAAA records), defaults to `ipv4 or ipv6`
- `"provider_ip"` can be set to `true` to let your DNS provider determine your IPv4 address (and/or IPv6 address) automatically when you send an update request, without sending the new IP address detected by the program in the request.

## Domain setup

[support.zoneedit.com/en/knowledgebase/article/dynamic-dns](https://support.zoneedit.com/en/knowledgebase/article/dynamic-dns)
