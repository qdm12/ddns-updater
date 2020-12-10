# Google

## Domain setup

Thanks to [@gauravspatel](https://github.com/gauravspatel) for #124

1. Enable dynamic DNS in the *synthetic records* section of DNS management.
1. The username and password is generated once you create the dynamic DNS entry.
1. If you want to create a **wildcard entry**, you have to create a custom CNAME record for the subdomain `"*"` which is pointed to `"@"`.
