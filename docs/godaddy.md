# GoDaddy

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "godaddy",
      "domain": "domain.com",
      "key": "key",
      "secret": "secret",
      "ip_version": "ipv4",
      "ipv6_suffix": ""
    }
  ]
}
```

### Compulsory parameters

- `"domain"` is the domain to update. It can be `example.com` (root domain), `sub.example.com` (subdomain of `example.com`) or `*.example.com` for the wildcard.
- `"key"`
- `"secret"`

### Optional parameters

- `"ip_version"` can be `ipv4` (A records), or `ipv6` (AAAA records) or `ipv4 or ipv6` (update one of the two, depending on the public ip found). It defaults to `ipv4 or ipv6`.
- `"ipv6_suffix"` is the IPv6 interface identifier suffix to use. It can be for example `0:0:0:0:72ad:8fbb:a54e:bedd/64`. If left empty, it defaults to no suffix and the raw public IPv6 address obtained is used in the record updating.

## Domain setup

[![GoDaddy Website](../readme/godaddy.png)](https://www.godaddy.com/en-ie)

1. Login to [https://developer.godaddy.com/keys](https://developer.godaddy.com/keys/) with your account credentials.

[![GoDaddy Developer Login](../readme/godaddy1.gif)](https://developer.godaddy.com/keys)

1. Generate a Test key and secret.

[![GoDaddy Developer Test Key](../readme/godaddy2.gif)](https://developer.godaddy.com/keys)

1. Generate a **Production** key and secret.

[![GoDaddy Developer Production Key](../readme/godaddy3.gif)](https://developer.godaddy.com/keys)

Obtain the **key** and **secret** of that production key.

In this example, the key is `dLP4WKz5PdkS_GuUDNigHcLQFpw4CWNwAQ5` and the secret is `GuUFdVFj8nJ1M79RtdwmkZ`.

## Testing

1. Go to [https://dcc.godaddy.com/manage/yourdomain.com/dns](https://dcc.godaddy.com/manage/yourdomain.com/dns) (replace yourdomain.com)

    [![GoDaddy DNS management](../readme/godaddydnsmanagement.png)](https://dcc.godaddy.com/manage/)

1. Change the IP address to `127.0.0.1`
1. Run the ddns-updater
1. Refresh the Godaddy webpage to check the update occurred.
