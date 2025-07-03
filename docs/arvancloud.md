# Arvancloud.ir

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "arvancloud",
      "domain": "sub.domain.com",
      "token": "apikey ..."
    }
  ]
}
```

### Compulsory parameters

- `"domain"` is the domain to update. It cannot be `example.com` (root domain) and should be like `sub.example.com` (subdomain of `example.com`).
- `"token"` like "apikey ...".

## Domain setup

- Create a policy for managing DNS in [Policies](https://panel.arvancloud.ir/profile/iam/policies)
- Create a token in [ArvanCloud profile](https://panel.arvancloud.ir/profile/iam/machine-users)
- Give access of the policy to the token
