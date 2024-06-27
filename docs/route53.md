# AWS

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "route53",
      "domain": "domain.com",
      "ip_version": "ipv4",
      "ipv6_suffix": "",
      "access_key": "ffffffffffffffffffff",
      "secret_key": "ffffffffffffffffffffffffffffffffffffffff",
      "zone_id": "A30888735ZF12K83Z6F00",
      "ttl": 300
    }
  ]
}
```

### Compulsory parameters

- `"domain"` is the domain to update. It can be `example.com` (root domain), `sub.example.com` (subdomain of `example.com`) or `*.example.com` for the wildcard.
- `"access_key"` is the `AWS_ACCESS_KEY`
- `"secret_key"` is the `AWS_SECRET_ACCESS_KEY`
- `"zone_id"` is identification of your hosted zone

### Optional parameters

- `"ip_version"` can be `ipv4` (A records), or `ipv6` (AAAA records) or `ipv4 or ipv6` (update one of the two, depending on the public ip found). It defaults to `ipv4 or ipv6`.
- `"ipv6_suffix"` is the IPv6 interface identifiersuffix to use. It can be for example `0:0:0:0:72ad:8fbb:a54e:bedd/64`. If left empty, it defaults to no suffix and the raw public IPv6 address obtained is used in the record updating.
- `"ttl"` amount of time, in seconds, that you want DNS recursive resolvers to cache information about this record. Defaults to `300`.

## Domain setup

Amazon has [an extensive documentation on registering or tranfering your domain to route53](https://docs.aws.amazon.com/Route53/latest/DeveloperGuide/Welcome.html).

### User permissions

Create a policy to grant access to change record sets, you can use a wildcard `*` in case you want to grant access to all your hosted zones.

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": "route53:ChangeResourceRecordSets",
      "Resource": "arn:aws:route53:::hostedzone/A30888735ZF12K83Z6F00"
    }
  ]
}
```
