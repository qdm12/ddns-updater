# GCP

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "gcp",
      "project": "my-project-id",
      "zone": "zone",
      "credentials": {
        "type": "service_account",
        "project_id": "my-project-id",
        // ...
      },
      "domain": "domain.com",
      "host": "@",
      "ip_version": "ipv4"
    }
  ]
}
```

### Compulsory parameters

- `"project"` is the id of your Google Cloud project
- `"zone"` is the zone, that your DNS record is located in
- `"credentials"` is the JSON credentials for your Google Cloud project. This is usually downloaded as a JSON file, which you can copy paste the content as the value of the `"credentials"` key. More information on how to get it is available [here](https://cloud.google.com/docs/authentication/getting-started). Please ensure your service account has all necessary permissions to create/update/list/get DNS records within your project.
- `"domain"` is the TLD of you DNS record (without a trailing dot)
- `"host"` is your host and can be a subdomain or `"@"` or `"*"`

### Optional parameters

- `"ip_version"` can be `ipv4` (A records) or `ipv6` (AAAA records), defaults to `ipv4`
