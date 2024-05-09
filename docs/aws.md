# AWS

## Configuration

### Example

<!-- UPDATE THIS JSON EXAMPLE -->

```json
{
  "settings": [
    {
      "provider": "aws",
      "domain": "domain.com",
      "host": "@",
      "ip_version": "ipv4",
      "ipv6_suffix": "",
      "aws_access_key_id": "ffffffffffffffffffff",
      "aws_secret_access_key": "ffffffffffffffffffffffffffffffffffffffff",
      "hosted_zone_id": "A30888735ZF12K83Z6F00",
      "region": "eu-central-1",
      "ttl": 60
    }
  ]
}
```

### Compulsory parameters

- `"domain"`
- `"host"` is your host and can be a subdomain or `"@"` or the wildcard `"*"`
- `"aws_access_key_id"`
- `"aws_secret_access_key"`
- `"hosted_zone_id"`

<!-- UPDATE THIS IF NEEDED -->

### Optional parameters

- `"ip_version"` can be `ipv4` (A records), or `ipv6` (AAAA records) or `ipv4 or ipv6` (update one of the two, depending on the public ip found). It defaults to `ipv4 or ipv6`.
- `"ipv6_suffix"` is the IPv6 interface identifiersuffix to use. It can be for example `0:0:0:0:72ad:8fbb:a54e:bedd/64`. If left empty, it defaults to no suffix and the raw public IPv6 address obtained is used in the record updating.
- `"ttl"` amount of time, in seconds, that you want DNS recursive resolvers to cache information about this record. Defaults to `300`.
- `"region"` of the route53 API. Route53 is a global resource, records created here will be globaly available unless you are using geolocation routing policy. Defaults to `us-east-1`.

<!-- UPDATE THIS IF NEEDED -->

## Domain setup

<!-- FILL THIS UP WITH A FEW NUMBERED STEPS -->
Amazon has an extensive documentation on registering or tranferring your domain to route53 <https://docs.aws.amazon.com/Route53/latest/DeveloperGuide/Welcome.html>. 

## User permissions

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
