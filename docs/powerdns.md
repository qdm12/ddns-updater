# PowerDNS

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "powerdns",
      "domain": "domain.com",
      "server_url": "http://powerdns.example.com:8081",
      "api_key": "your-api-key",
      "server_id": "localhost",
      "ttl": 300,
      "ip_version": "ipv4",
      "ipv6_suffix": ""
    }
  ]
}
```

### Compulsory parameters

- `"domain"` is the domain to update. It can be `example.com` (root domain), `sub.example.com` (subdomain of `example.com`) or `*.example.com` for the wildcard.
- `"server_url"` is the URL of your PowerDNS Authoritative server HTTP API (e.g. `http://powerdns.example.com:8081`)
- `"api_key"` is the API key for authentication, configured in your PowerDNS server's `api-key` setting

### Optional parameters

- `"server_id"` is the PowerDNS server ID. It defaults to `localhost`.
- `"ttl"` optional integer value corresponding to a number of seconds. It defaults to `300`.
- `"ip_version"` can be `ipv4` (A records), or `ipv6` (AAAA records) or `ipv4 or ipv6` (update one of the two, depending on the public ip found). It defaults to `ipv4 or ipv6`.
- `"ipv6_suffix"` is the IPv6 interface identifier suffix to use. It can be for example `0:0:0:0:72ad:8fbb:a54e:bedd/64`. If left empty, it defaults to no suffix and the raw public IPv6 address obtained is used in the record updating.

## PowerDNS setup

1. Enable the HTTP API in your PowerDNS Authoritative server configuration:

   ```
   api=yes
   api-key=your-api-key
   webserver=yes
   webserver-address=0.0.0.0
   webserver-port=8081
   webserver-allow-from=0.0.0.0/0
   ```

2. Make sure the zone for your domain already exists in PowerDNS before using ddns-updater.
