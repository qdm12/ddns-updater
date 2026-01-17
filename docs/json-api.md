# JSON API Endpoint

The DDNS Updater provides a JSON API endpoint at `/json` that returns the same information as the web interface but in JSON format.

## Endpoint

- **URL**: `/json`
- **Method**: `GET`
- **Content-Type**: `application/json`
- **Cache-Control**: `no-cache`

## Response Format

The response is a JSON object with the following structure:

```json
{
  "records": [
    {
      "domain": "example.com",
      "owner": "www",
      "provider": "cloudflare",
      "ip_version": "ipv4",
      "status": "success",
      "message": "no IP change for 2h",
      "current_ip": "192.168.1.1",
      "previous_ips": ["192.168.1.2", "192.168.1.3"],
      "total_ips_in_history": 3,
      "last_update": "2023-01-01T12:00:00Z",
      "success_time": "2023-01-01T12:00:00Z",
      "duration_since_success": "2h"
    }
  ],
  "time": "2023-01-01T12:00:00Z",
  "last_success_time": "2023-01-01T12:00:00Z",
  "last_success_ip": "192.168.1.1"
}
```

## Field Descriptions

### Root Object
- `records`: Array of DNS record objects
- `time`: Current server time when the response was generated
- `last_success_time`: Time of the most recent successful update across all records (zero time if no successful updates)
- `last_success_ip`: IP address from the record with the most recent successful update (empty string if no successful updates)

### Record Object
- `domain`: The domain name (e.g., "example.com")
- `owner`: The subdomain owner (e.g., "www")
- `provider`: The DNS provider name (e.g., "cloudflare", "duckdns")
- `ip_version`: IP version supported ("ipv4", "ipv6", or "ipv4 or ipv6")
- `status`: Current status ("success", "failure", "updating", "up to date", "unset")
- `message`: Additional status information or error message
- `current_ip`: Current IP address (or "N/A" if not available)
- `previous_ips`: Array of previous IP addresses (limited to last 10, empty if none)
- `total_ips_in_history`: Total number of IP addresses available in the history
- `last_update`: Timestamp of the last update attempt
- `success_time`: Timestamp of the last successful update
- `duration_since_success`: Human-readable duration since last success (e.g., "2h", "30m", "1d")

## Example Usage

### Using curl
```bash
curl http://localhost:8080/json
```

### Using JavaScript
```javascript
fetch('/json')
  .then(response => response.json())
  .then(data => {
    console.log('Current time:', data.time);
    data.records.forEach(record => {
      console.log(`${record.domain} (${record.provider}): ${record.status}`);
    });
  });
```

### Using Python
```python
import requests

response = requests.get('http://localhost:8080/json')
data = response.json()

print(f"Current time: {data['time']}")
for record in data['records']:
    print(f"{record['domain']} ({record['provider']}): {record['status']}")
```

## Status Values

- `success`: Last update was successful
- `failure`: Last update failed
- `updating`: Update is currently in progress
- `up to date`: No IP change needed
- `unset`: Status not yet determined
