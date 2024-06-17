# Mikrotik

## Configuration

### Example

```json
{
  "settings": [
    {
      "provider": "mikrotik",
      "router_ip": "192.168.0.1",
      "address_list": "AddressListName",
      "username": "user",
      "password": "secret",
      "ip_version": "ipv4"
    }
  ]
}
```

### Parameters

- `"router_ip"` is the IP address of your router
- `"address_list"` is the name of the address list
- `"username"` is the username to authenticate with
- `"password"` is the user's password

## Domain setup

- Create a user with read, write, and api access
- Optionally create an entry in `/ip firewall address-list` to assign your public IP, an entry will be created for you otherwise
- You can then use this address list in your hairpin NAT firewall rules
