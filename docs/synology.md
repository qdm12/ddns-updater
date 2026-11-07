# Synology

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "synology",
      "domain": "X.synology.me",
      "myds_id": "",
      "serial": "",
      "mac_address": "00:11:32:00:00:00",
      "auth_key": "",
      "api_key": ""
    }
  ]
}
```

### Compulsory parameters

- `"domain"` is the domain to update. `NAME.synology.me` or other registered domains by Synology.
- `"api_key"` Can be obtained by executing following command on Synology server.
  ```bash
  synocloudserviceauth -a get
  ```
- Execute following command on Synology server.
  ```bash
  synowebapi --exec api=SYNO.Core.Package.MyDS method=get
  ```

  Copy `"auth_key"` `"auth_key"` and `"myds_id"` into config file.

### Optional parameters

- `"mac_address"` Network card mac address of synology server. It's optional as code will try to determine current network mac address (on machine where ddns-updates is executed).
