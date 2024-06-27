# Custom provider

The custom provider allows to configure a URL with a few additional parameters to update your records.

For now it sends an HTTP GET request to the URL given with some additional parameters.
Feel free to open issues to extend its configuration options.

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "custom",
      "domain": "example.com",
      "url": "https://example.com/update?domain=example.com&host=@&username=username&client_key=client_key",
      "ipv4key": "ipv4",
      "ipv6key": "ipv6",
      "success_regex": "good",
      "ip_version": "ipv4",
      "ipv6_suffix": ""
    }
  ]
}
```

### Compulsory parameters

- `"domain"` is the domain to update. It can be `example.com` (root domain), `sub.example.com` (subdomain of `example.com`) or `*.example.com` for the wildcard.
- `"url"` is the URL to update your records and should contain all the information EXCEPT the IP address to update
- `"ipv4key"` is the URL query parameter name for the IPv4 address, for example `ipv4` will be added to the URL with `&ipv4=1.2.3.4`.
- `"ipv6key"` is the URL query parameter name for the IPv6 address, for example `ipv6` will be added to the URL with `&ipv6=::aaff`. Even if you don't use IPv6, this must be set to something.
- `"success_regex"` is a regular expression to match the response from the server to determine if the update was successful. You can use [regex101.com](https://regex101.com/) to find the regular expression you want. For example `good` would match any response containing the word "good".

### Optional parameters

- `"ip_version"` can be `ipv4` (A records), or `ipv6` (AAAA records) or `ipv4 or ipv6` (update one of the two, depending on the public ip found). It defaults to `ipv4 or ipv6`.
- `"ipv6_suffix"` is the IPv6 interface identifier suffix to use. It can be for example `0:0:0:0:72ad:8fbb:a54e:bedd/64`. If left empty, it defaults to no suffix and the raw public IPv6 address obtained is used in the record updating.
