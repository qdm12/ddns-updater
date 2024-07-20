# Azure

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "azure",
      "domain": "domain.com",
      "host": "@",
      "tenant_id": "",
      "client_id": "",
      "client_secret": "",
      "subscription_id": "",
      "resource_group_name": ""
    }
  ]
}
```

### Compulsory parameters

- `"domain"`
- `"host"`
- `"tenant_id"`
- `"client_id"`
- `"client_secret"`
- `"subscription_id"` found in the properties section of Azure DNS
- `"resource_group_name"` found in the properties section of Azure DNS

### Optional parameters

- `"ip_version"` can be `ipv4` (A records) or `ipv6` (AAAA records), defaults to `ipv4 or ipv6`

## Domain setup

Thanks to @danimart1991 for describing the following steps!

1. Create Domain
1. Activate Azure DNS Zone for that domain
1. Find the following parameters in the Properties section of Azure DNS:
    - The name or URL `AnyNameOrUrl` for the query below **TODO**
    - `subscription_id`
    - `resource_group_name`
1. In the Azure Console (inside the portal), run:

    ```sh
    az ad sp create-for-rbac -n "$AnyNameOrUrl" --scopes "/subscriptions/$subscription_id/resourceGroups/$resource_group_name/providers/Microsoft.Network/dnszones/$zone_name"
    ```

    This gives you the rest of the parameters:

    ```json
    {
      "appId": "{app_id/client_id}",
      "displayName": "not important",
      "name": "not important",
      "password": "{app_password}",
      "tenant": "not important"
    }
    ```
