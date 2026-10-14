# HTTP API

## Update DNS records

Send a `POST` request to `/update` to start an immediate update of all configured
DNS records:

```sh
curl -X POST http://localhost:8000/update
```

The endpoint used to accept `GET` requests. It now requires `POST` because an
update changes external DNS state. A `GET /update` request returns `405 Method
Not Allowed` with an `Allow: POST` response header.

When `ROOT_URL` is set, prepend its path to the endpoint. For example, with
`ROOT_URL=/ddns`, send the request to `/ddns/update`.
