# Lightweight universal DDNS Updater with Docker and web UI

*Light container updating DNS A and/or AAAA records periodically for multiple DNS providers*

<img height="200" alt="DDNS Updater logo" src="https://raw.githubusercontent.com/qdm12/ddns-updater/master/readme/ddnsgopher.svg?sanitize=true">

[![Build status](https://github.com/qdm12/ddns-updater/workflows/Buildx%20latest/badge.svg)](https://github.com/qdm12/ddns-updater/actions?query=workflow%3A%22Buildx+latest%22)
[![Docker Pulls](https://img.shields.io/docker/pulls/qmcgaw/ddns-updater.svg)](https://hub.docker.com/r/qmcgaw/ddns-updater)
[![Docker Stars](https://img.shields.io/docker/stars/qmcgaw/ddns-updater.svg)](https://hub.docker.com/r/qmcgaw/ddns-updater)
[![Image size](https://images.microbadger.com/badges/image/qmcgaw/ddns-updater.svg)](https://microbadger.com/images/qmcgaw/ddns-updater)
[![Image version](https://images.microbadger.com/badges/version/qmcgaw/ddns-updater.svg)](https://microbadger.com/images/qmcgaw/ddns-updater)

[![Join Slack channel](https://img.shields.io/badge/slack-@qdm12-yellow.svg?logo=slack)](https://join.slack.com/t/qdm12/shared_invite/enQtODMwMDQyMTAxMjY1LTU1YjE1MTVhNTBmNTViNzJiZmQwZWRmMDhhZjEyNjVhZGM4YmIxOTMxOTYzN2U0N2U2YjQ2MDk3YmYxN2NiNTc)
[![GitHub last commit](https://img.shields.io/github/last-commit/qdm12/ddns-updater.svg)](https://github.com/qdm12/ddns-updater/issues)
[![GitHub commit activity](https://img.shields.io/github/commit-activity/y/qdm12/ddns-updater.svg)](https://github.com/qdm12/ddns-updater/issues)
[![GitHub issues](https://img.shields.io/github/issues/qdm12/ddns-updater.svg)](https://github.com/qdm12/ddns-updater/issues)

## Features

- Updates periodically A records for different DNS providers:
    - Cloudflare
    - DDNSS.de
    - DigitalOcean
    - DonDominio
    - DNSOMatic
    - DNSPod
    - Dreamhost
    - DuckDNS
    - DynDNS
    - FreeDNS
    - Gandi
    - GoDaddy
    - Google
    - He.net
    - Infomaniak
    - Linode
    - LuaDNS
    - Namecheap
    - NoIP
    - OpenDNS
    - OVH
    - Selfhost.de
    - Strato.de
    - **Want more?** [Create an issue for it](https://github.com/qdm12/ddns-updater/issues/new/choose)!
- Web User interface

![Web UI](https://raw.githubusercontent.com/qdm12/ddns-updater/master/readme/webui.png)

- 14MB Docker image based on a Go static binary in a Scratch Docker image with ca-certificates and timezone data
- Persistence with a JSON file *updates.json* to store old IP addresses with change times for each record
- Docker healthcheck verifying the DNS resolution of your domains
- Highly configurable
- Sends notifications to your Android phone, see the [**Gotify**](#Gotify) section (it's free, open source and self hosted 🆒)
- Compatible with `amd64`, `386`, `arm64`, `arm32v7` (Raspberry Pis) CPU architectures.

## Setup

The program reads the configuration from a JSON object, either from a file or from an environment variable.

1. Create a directory of your choice, say *data* with a file named **config.json** inside:

    ```sh
    mkdir data
    touch data/config.json
    # Owned by user ID of Docker container (1000)
    chown -R 1000 data
    # all access (for creating json database file data/updates.json)
    chmod 700 data
    # read access only
    chmod 400 data/config.json
    ```

    *(You could change the user ID, for example with `1001`, by running the container with `--user=1001`)*

1. Write a JSON configuration in *data/config.json*, for example:

    ```json
    {
        "settings": [
            {
                "provider": "namecheap",
                "domain": "example.com",
                "host": "@",
                "password": "e5322165c1d74692bfa6d807100c0310"
            }
        ]
    }
    ```

    You can find more information in the [configuration section](#configuration) to customize it.

1. Run the container with

    ```sh
    docker run -d -p 8000:8000/tcp -v "$(pwd)"/data:/updater/data qmcgaw/ddns-updater
    ```

1. (Optional) You can also set your JSON configuration as a single environment variable line (i.e. `{"settings": [{"provider": "namecheap", ...}]}`), which takes precedence over config.json. Note however that if you don't bind mount the `/updater/data` directory, there won't be a persistent database file `/updater/updates.json` but it will still work.

### Next steps

You can also use [docker-compose.yml](https://github.com/qdm12/ddns-updater/blob/master/docker-compose.yml) with:

```sh
docker-compose up -d
```

You can update the image with `docker pull qmcgaw/ddns-updater`. Other [Docker image tags are available](https://hub.docker.com/repository/docker/qmcgaw/ddns-updater/tags).

## Configuration

Start by having the following content in *config.json*, or in your `CONFIG` environment variable:

```json
{
    "settings": [
        {
            "provider": "",
        },
        {
            "provider": "",
        }
    ]
}
```

For each setting, you need to fill in parameters.
Check the documentation for your DNS provider:

- [Cloudflare](https://github.com/qdm12/ddns-updater/blob/master/docs/cloudflare.md)
- [DDNSS.de](https://github.com/qdm12/ddns-updater/blob/master/docs/ddnss.de.md)
- [DigitalOcean](https://github.com/qdm12/ddns-updater/blob/master/docs/digitalocean.md)
- [DonDominio](https://github.com/qdm12/ddns-updater/blob/master/docs/dondominio.md)
- [DNSOMatic](https://github.com/qdm12/ddns-updater/blob/master/docs/dnsomatic.md)
- [DNSPod](https://github.com/qdm12/ddns-updater/blob/master/docs/dnspod.md)
- [Dreamhost](https://github.com/qdm12/ddns-updater/blob/master/docs/dreamhost.md)
- [DuckDNS](https://github.com/qdm12/ddns-updater/blob/master/docs/duckdns.md)
- [DynDNS](https://github.com/qdm12/ddns-updater/blob/master/docs/dyndns.md)
- [DynV6](https://github.com/qdm12/ddns-updater/blob/master/docs/dynv6.md)
- [FreeDNS](https://github.com/qdm12/ddns-updater/blob/master/docs/freedns.md)
- [Gandi](https://github.com/qdm12/ddns-updater/blob/master/docs/gandi.md)
- [GoDaddy](https://github.com/qdm12/ddns-updater/blob/master/docs/godaddy.md)
- [Google](https://github.com/qdm12/ddns-updater/blob/master/docs/google.md)
- [He.net](https://github.com/qdm12/ddns-updater/blob/master/docs/he.net.md)
- [Infomaniak](https://github.com/qdm12/ddns-updater/blob/master/docs/infomaniak.md)
- [Linode](https://github.com/qdm12/ddns-updater/blob/master/docs/linode.md)
- [LuaDNS](https://github.com/qdm12/ddns-updater/blob/master/docs/luadns.md)
- [Namecheap](https://github.com/qdm12/ddns-updater/blob/master/docs/namecheap.md)
- [NoIP](https://github.com/qdm12/ddns-updater/blob/master/docs/noip.md)
- [OpenDNS](https://github.com/qdm12/ddns-updater/blob/master/docs/opendns.md)
- [OVH](https://github.com/qdm12/ddns-updater/blob/master/docs/ovh.md)
- [Selfhost.de](https://github.com/qdm12/ddns-updater/blob/master/docs/selfhost.de.md)
- [Strato.de](https://github.com/qdm12/ddns-updater/blob/master/docs/strato.md)

Note that:

- you can specify multiple hosts for the same domain using a comma separated list. For example with `"host": "@,subdomain1,subdomain2",`.

### Environment variables

| Environment variable | Default | Description |
| --- | --- | --- |
| `CONFIG` | | One line JSON object containing the entire config (takes precendence over config.json file) if specified |
| `PERIOD` | `5m` | Default period of IP address check, following [this format](https://golang.org/pkg/time/#ParseDuration) |\
| `PUBLICIP_FETCHERS` | `all` | Comma separated fetcher types to obtain the public IP address from `http` and `dns` |
| `PUBLICIP_HTTP_PROVIDERS` | `all` | Comma separated providers to obtain the public IP address (ipv4 or ipv6). See the [Public IP section](#Public-IP) |
| `PUBLICIPV4_HTTP_PROVIDERS` | `all` | Comma separated providers to obtain the public IPv4 address only. See the [Public IP section](#Public-IP) |
| `PUBLICIPV6_HTTP_PROVIDERS` | `all` | Comma separated providers to obtain the public IPv6 address only. See the [Public IP section](#Public-IP) |
| `PUBLICIP_DNS_PROVIDERS` | `all` | Comma separated providers to obtain the public IP address (IPv4 and/or IPv6). See the [Public IP section](#Public-IP) |
| `UPDATE_COOLDOWN_PERIOD` | `5m` | Duration to cooldown between updates for each record. This is useful to avoid being rate limited or banned. |
| `HTTP_TIMEOUT` | `10s` | Timeout for all HTTP requests |
| `LISTENING_PORT` | `8000` | Internal TCP listening port for the web UI |
| `ROOT_URL` | `/` | URL path to append to all paths to the webUI (i.e. `/ddns` for accessing `https://example.com/ddns` through a proxy) |
| `BACKUP_PERIOD` | `0` | Set to a period (i.e. `72h15m`) to enable zip backups of data/config.json and data/updates.json in a zip file |
| `BACKUP_DIRECTORY` | `/updater/data` | Directory to write backup zip files to if `BACKUP_PERIOD` is not `0`. |
| `LOG_LEVEL` | `info` | Level of logging, `info`, `warning` or `error` |
| `LOG_CALLER` | `hidden` | Show caller per log line, `hidden` or `short` |
| `GOTIFY_URL` |  | (optional) HTTP(s) URL to your Gotify server |
| `GOTIFY_TOKEN` |  | (optional) Token to access your Gotify server |
| `TZ` | | Timezone to have accurate times, i.e. `America/Montreal` |

#### Public IP

By default, all public IP fetching types are used and cycled (over DNS and over HTTPs).

On top of that, for each fetching method, all echo services available are cycled on each request.

This allows you not to be blocked for making too many requests.

You can otherwise customize it with the following:

- `PUBLICIP_HTTP_PROVIDERS` gets your public IPv4 or IPv6 address. It can be one or more of the following:
  - `opendns` using [https://diagnostic.opendns.com/myip](https://diagnostic.opendns.com/myip)
  - `ifconfig` using [https://ifconfig.io/ip](https://ifconfig.io/ip)
  - `ipinfo` using [https://ipinfo.io/ip](https://ipinfo.io/ip)
  - `ddnss` using [https://ddnss.de/meineip.php](https://ddnss.de/meineip.php)
  - `google` using [https://domains.google.com/checkip](https://domains.google.com/checkip)
  - You can also specify an HTTPS URL such as `https://ipinfo.io/ip`
- `PUBLICIPV4_HTTP_PROVIDERS` gets your public IPv4 address only. It can be one or more of the following:
  - `ipify` using [https://api.ipify.org](https://api.ipify.org)
  - `noip` using [http://ip1.dynupdate.no-ip.com](http://ip1.dynupdate.no-ip.com)
  - You can also specify an HTTPS URL such as `https://ipinfo.io/ip`
- `PUBLICIPV6_HTTP_PROVIDERS` gets your public IPv6 address only. It can be one or more of the following:
  - `ipify` using [https://api6.ipify.org](https://api6.ipify.org)
  - `noip` using [http://ip1.dynupdate6.no-ip.com](http://ip1.dynupdate6.no-ip.com)
  - You can also specify an HTTPS URL such as `https://ipinfo.io/ip`
- `PUBLICIP_DNS_PROVIDERS` gets your public IPv4 address only or IPv6 address only or one of them (see #136). It can be one or more of the following:
  - `google`
  - `cloudflare`

### Host firewall

If you have a host firewall in place, this container needs the following ports:

- TCP 443 outbound for outbound HTTPS
- TCP 80 outbound if you use a local unsecured HTTP connection to your Gotify server
- UDP 53 outbound for outbound DNS resolution
- TCP 8000 inbound (or other) for the WebUI

## Architecture

At program start and every period (5 minutes by default):

1. Fetch your public IP address
1. For each record:
    1. DNS resolve it to obtain its current IP address(es)
        - If the resolution fails, update the record with your public IP address by calling the DNS provider API and finish
    1. Check if your public IP address is within the resolved IP addresses
        - Yes: skip the update
        - No: update the record with your public IP address by calling the DNS provider API

💡 We do DNS resolution every period so it detects a change made to the record manually, for example on the DNS provider web UI
💡 As DNS resolutions are essentially free and without rate limiting, these are great to avoid getting banned for too many requests.

### Special case: Cloudflare

For Cloudflare records with the `proxied` option, the following is done.

At program start and every period (5 minutes by default), for each record:

1. Fetch your public IP address
1. For each record:
    1. Check the last IP address (persisted in `updates.json`) for that record
        - If it doesn't exist, update the record with your public IP address by calling the DNS provider API and finish
    1. Check if your public IP address matches the last IP address you updated the record with
        - Yes: skip the update
        - No: update the record with your public IP address by calling the DNS provider API

This is the only way as doing a DNS resolution on the record will give the IP address of a Cloudflare server instead of your server.

⚠️ This has the disadvantage that if the record is changed manually, the program will not detect it.
We could do an API call to get the record IP address every period, but that would get you banned especially with a low period duration.

## Gotify

[![Gotify](https://github.com/qdm12/ddns-updater/blob/master/readme/gotify.png?raw=true)](https://gotify.net)

[**Gotify**](https://gotify.net) is a simple server for sending and receiving messages, and it is **free**, **private** and **open source**

- It has an [Android app](https://play.google.com/store/apps/details?id=com.github.gotify) to receive notifications
- The app does not drain your battery 👍
- The notification server is self hosted, see [how to set it up with Docker](https://gotify.net/docs/install)
- The notifications only go through your own server (ideally through HTTPS though)

To set it up with DDNS updater:

1. Go to the Web GUI of Gotify
1. Login with the admin credentials
1. Create an app and copy the generated token to the environment variable `GOTIFYTOKEN` (for this container)
1. Set the `GOTIFYURL` variable to the URL of your Gotify server address (i.e. `http://127.0.0.1:8080` or `https://bla.com/gotify`)

## Testing

- The automated healthcheck verifies all your records are up to date [using DNS lookups](https://github.com/qdm12/ddns-updater/blob/master/internal/healthcheck/healthcheck.go#L15)
- You can also manually check, by:
    1. Going to your DNS management webpage
    1. Setting your record to `127.0.0.1`
    1. Run the container
    1. Refresh the DNS management webpage and verify the update happened

## Development and contributing

- [Contribute with code](https://github.com/qdm12/ddns-updater/blob/master/docs/contributing.md)
- [Github workflows to know what's building](https://github.com/qdm12/ddns-updater/actions)
- [List of issues and feature requests](https://github.com/qdm12/ddns-updater/issues)
- [Kanban board](https://github.com/qdm12/ddns-updater/projects/1)

## License

This repository is under an [MIT license](https://github.com/qdm12/ddns-updater/master/license)

## Used in external projects

- [Starttoaster/docker-traefik](https://github.com/Starttoaster/docker-traefik#home-networks-extra-credit-dynamic-dns)

## Support

Sponsor me on [Github](https://github.com/sponsors/qdm12) or donate to [paypal.me/qmcgaw](https://www.paypal.me/qmcgaw)

[![https://github.com/sponsors/qdm12](https://raw.githubusercontent.com/qdm12/private-internet-access-docker/master/doc/sponsors.jpg)](https://github.com/sponsors/qdm12)
[![https://www.paypal.me/qmcgaw](https://raw.githubusercontent.com/qdm12/private-internet-access-docker/master/doc/paypal.jpg)](https://www.paypal.me/qmcgaw)

Many thanks to J. Famiglietti for supporting me financially 🥇👍
