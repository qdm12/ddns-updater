# Aliyun

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "aliyun",
      "domain": "domain.com",
      "host": "@",
      "ip_version": "ipv4",
      "region_id": "regionId",
      "access_key_id": "accessKeyId",
      "access_key_secret": "accessKeySecret"
    }
  ]
}
```

### Compulsory parameters

- `"domain"`
- `"host"` is your host and can be a subdomain or `"@"`
- `"region_id"` is your region ID
- `"access_key_id"` is your AccessKey ID
- `"access_key_secret"` is your AccessKey Secret

### Optional parameters

- `"ip_version"` can be `ipv4` (A records) or `ipv6` (AAAA records), defaults to `ipv4 or ipv6`

## Domain setup