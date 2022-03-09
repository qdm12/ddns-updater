# Cloudflare

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "cloudflare",
      "zone_identifier": "some id",
      "identifier": "<IDENTIFIER>",
      "domain": "domain.com",
      "host": "@",
      "ttl": 600,
      "token": "yourtoken",
      "ip_version": "ipv4"
    }
  ]
}
```

### Compulsory parameters

- `"zone_identifier"` is the Zone ID of your site
- `"identifier"` is the ID of your API Token (Step 4 in <b>Domain setup</b>)
- `"domain"`
- `"host"` is your host. It should be left to `"@"`, since subdomain and wildcards (`"*"`) are not really supported by Cloudflare it seems.
See [this issue comment for context](https://github.com/qdm12/ddns-updater/issues/243#issuecomment-928313949). This is left as is for compatibility.
- `"ttl"` integer value for record TTL in seconds (specify 1 for automatic)
- One of the following:
    - Email `"email"` and Global API Key `"key"`
    - User service key `"user_service_key"`
    - API Token `"token"`, configured with DNS edit permissions for your DNS name's zone

### Optional parameters

- `"proxied"` can be set to `true` to use the proxy services of Cloudflare
- `"ip_version"` can be `ipv4` (A records) or `ipv6` (AAAA records), and defaults to `ipv4 or ipv6`

## Domain setup

1. Make sure you have `curl` installed
2. Obtain your API key from Cloudflare website ([see this](https://support.cloudflare.com/hc/en-us/articles/200167836-Where-do-I-find-my-Cloudflare-API-key-))
3. Obtain your zone identifier for your domain name, from the domain's overview page written as *Zone ID*
4. Find your **identifier** in the `id` field with

    ```sh
      curl -X GET "https://api.cloudflare.com/client/v4/user/tokens/verify" \
           -H "Authorization: Bearer <API_TOKNEN>" \
           -H "Content-Type:application/json"
    ```

    ```
    Output:
        {
         "result":{
            "id":"<identifier>",    <-------- This is the identifier.
            "status":"active"
         },
         "success":true,
         "errors":[

         ],
         "messages":[
            {
               "code":10000,
               "message":"This API Token is valid and active",
               "type":null
            }
         ]
      }
    ```

You can now fill in the necessary parameters in *config.json*

Special thanks to @Starttoaster for helping out with the [documentation](https://gist.github.com/Starttoaster/07d568c2a99ad7631dd776688c988326) and testing.
