# ClouDNS

## Configuration

### IPv4 example

```json
{
  "settings": [
    {
      "provider": "cloudns",
      "domain": "home.example.com",
      "url": "https://ipv4.cloudns.net/api/dynamicURL/?q=YOUR_UNIQUE_KEY",
      "ip_version": "ipv4"
    }
  ]
}
```

### IPv6 example

```json
{
  "settings": [
    {
      "provider": "cloudns",
      "domain": "home.example.com",
      "url": "https://ipv6.cloudns.net/api/dynamicURL/?q=YOUR_IPV6_RECORD_KEY",
      "ip_version": "ipv6"
    }
  ]
}
```

### Compulsory parameters

- `"domain"` is the domain to update, such as `example.com`, `home.example.com` or `*.example.com`.
- `"url"` is the HTTPS Dynamic URL generated for that specific DNS record. Use the standard URL containing only the `q` token, without additional parameters. Only `ipv4.cloudns.net/api/dynamicURL/` and `ipv6.cloudns.net/api/dynamicURL/` are supported.
- `"ip_version"` must explicitly be `ipv4` for an A record and the IPv4 endpoint, or `ipv6` for an AAAA record and the IPv6 endpoint. The default `ipv4 or ipv6` is not supported.

### Optional parameters

- `"ipv6_suffix"` is the common IPv6 interface identifier suffix setting. It retains its usual updater behavior, but cannot control the address ClouDNS writes: ClouDNS uses the request's source address independently. Leave it empty when using the machine's outbound IPv6 address.

## Domain setup

1. Sign in to [ClouDNS](https://www.cloudns.net/) and open your DNS zone.
1. Create or select the A record to update (or an AAAA record for IPv6).
1. Click the arrows icon for **Active Dynamic URL** next to the record, then activate it.
1. Copy the generated Dynamic URL for that record.
1. Paste it into `url` in a setting with `provider` set to `cloudns` and the corresponding `domain`.
1. Select `ip_version` to match the record and URL endpoint, as shown above.

Each Dynamic URL is a secret credential for one record. Multiple records, including A and AAAA records for the same name, require their own URLs and separate settings. No account API credentials are required. Never share these URLs or commit them to a public repository.

ClouDNS documents the [IPv4 and IPv6 endpoints](https://www.cloudns.net/wiki/article/312/) and the [standard `OK` success response](https://www.cloudns.net/wiki/article/364/). The updater sends an HTTPS GET to the configured URL unchanged and accepts only `OK` (ignoring surrounding whitespace) as success.

## Source address and logging considerations

ClouDNS determines the record's IP automatically from the incoming HTTP connection. The updater does not transmit its detected IP address. Proxies, VPNs, NAT, and different outbound network paths can therefore cause ClouDNS to detect a different address from the updater's public IP detection. IPv6 requires working outbound IPv6 connectivity to the IPv6 endpoint; an IPv6 suffix cannot make ClouDNS update a different device's address.

After success, the updater records its own detected IP in its update history to comply with the existing provider interface. The `OK` response does not confirm which address ClouDNS wrote. Ensure both detection and update traffic use the intended outbound network path.

Existing debug logging includes full HTTP request URLs and configuration contents, exposing the secret `q` token. Avoid debug logging with production credentials, and redact configuration and logs before sharing them.
