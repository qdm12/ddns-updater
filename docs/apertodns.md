# ApertoDNS

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "apertodns",
      "domain": "apertodns.com",
      "owner": "home",
      "token": "apertodns_live_xxxxxxxxxxxxx",
      "ip_version": "ipv4",
      "ipv6_suffix": ""
    }
  ]
}
```

### Compulsory parameters

- `"provider"`: `"apertodns"`
- `"domain"`: Your ApertoDNS domain (e.g., `"apertodns.com"` or your custom domain)
- `"owner"`: The subdomain/hostname (e.g., `"home"`, `"office"`, `"@"` for root)
- `"token"`: Your ApertoDNS API token (starts with `apertodns_live_` or `apertodns_test_`)

### Optional parameters

- `"ip_version"`: `"ipv4"` (A records), `"ipv6"` (AAAA records), or `"ipv4 or ipv6"`
- `"ipv6_suffix"`: IPv6 suffix for EUI-64 addressing
- `"base_url"`: Custom API endpoint (default: `"https://api.apertodns.com"`)

## Domain setup

1. Sign up at [apertodns.com](https://apertodns.com)
2. Create a domain in your dashboard
3. Generate an API token from Settings -> API Keys
4. Use the token in your configuration

## Protocol Support

This provider implements the full [ApertoDNS Protocol v1.2](https://docs.apertodns.com/protocol) with automatic fallback:

1. **Modern API** (primary): `POST /.well-known/apertodns/v1/update`
   - Bearer token authentication
   - JSON request/response

2. **Legacy DynDNS2** (fallback): `GET /nic/update`
   - Basic authentication
   - Text response ("good", "nochg")

The provider automatically uses the modern API first. If the server doesn't
support the modern endpoint (404) or has a temporary error (500), it falls
back to the legacy DynDNS2 endpoint.

**Note**: Authentication errors (invalid token, hostname not found) do NOT
trigger fallback, as these would fail on both endpoints.

## Modern API Endpoint

```
POST {base_url}/.well-known/apertodns/v1/update
Authorization: Bearer {token}
Content-Type: application/json

{
  "hostname": "home.apertodns.com",
  "ipv4": "93.44.241.82"
}
```

### Response (Success)

```json
{
  "success": true,
  "data": {
    "hostname": "home.apertodns.com",
    "ipv4": "93.44.241.82",
    "ipv6": null,
    "ttl": 300,
    "updated_at": "2025-01-02T12:00:00.000Z"
  }
}
```

### Response (Error)

```json
{
  "success": false,
  "error": {
    "code": "invalid_token",
    "message": "Invalid or expired token"
  }
}
```

## Legacy DynDNS2 Endpoint

```
GET {base_url}/nic/update?hostname={hostname}&myip={ip}
Authorization: Basic base64(token:{token})
```

### Response

```
good 93.44.241.82
nochg 93.44.241.82
```

## Error Codes

| Code | HTTP Status | Meaning |
|------|-------------|---------|
| `invalid_token` | 401 | Invalid or expired token |
| `unauthorized` | 401 | Missing authentication |
| `hostname_not_found` | 404 | Hostname not found |
| `invalid_ip` | 400 | Invalid IP address format |
| `rate_limited` | 429 | Rate limit exceeded |

## Custom Server

ApertoDNS is an open protocol. You can use any compatible server by setting the `base_url` parameter:

```json
{
  "settings": [
    {
      "provider": "apertodns",
      "base_url": "https://ddns.example.com",
      "domain": "example.com",
      "owner": "home",
      "token": "your_token_here"
    }
  ]
}
```

## Links

- Website: [apertodns.com](https://apertodns.com)
- Documentation: [docs.apertodns.com](https://docs.apertodns.com)
- Protocol Spec: [ApertoDNS Protocol v1.2](https://docs.apertodns.com/protocol)
