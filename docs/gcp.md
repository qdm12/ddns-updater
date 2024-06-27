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
      "ip_version": "ipv4",
      "ipv6_suffix": ""
    }
  ]
}
```

### Compulsory parameters

- `"project"` is the id of your Google Cloud project
- `"zone"` is the zone, that your DNS record is located in
- `"credentials"` is the JSON credentials for your Google Cloud project. This is usually downloaded as a JSON file, which you can copy paste the content as the value of the `"credentials"` key. More information on how to get it is available [here](https://cloud.google.com/docs/authentication/getting-started). Please ensure your service account has all necessary permissions to create/update/list/get DNS records within your project.
- `"domain"` is the domain to update. It can be `example.com` (root domain), `sub.example.com` (subdomain of `example.com`) or `*.example.com` for the wildcard.

### Optional parameters

- `"ip_version"` can be `ipv4` (A records), or `ipv6` (AAAA records) or `ipv4 or ipv6` (update one of the two, depending on the public ip found). It defaults to `ipv4 or ipv6`.
- `"ipv6_suffix"` is the IPv6 interface identifier suffix to use. It can be for example `0:0:0:0:72ad:8fbb:a54e:bedd/64`. If left empty, it defaults to no suffix and the raw public IPv6 address obtained is used in the record updating.
