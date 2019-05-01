# Lightweight DDNS Updater with Docker and web UI

*Light container updating DNS A records periodically for GoDaddy, Namecheap, Dreamhost and DuckDNS*

**WARNING: Please change your configuration to use *config.json*, see the [config.json section](#configuration)**

[![DDNS Updater by Quentin McGaw](https://github.com/qdm12/ddns-updater/raw/master/readme/title.png)](https://hub.docker.com/r/qmcgaw/ddns-updater)

[![Docker Build Status](https://img.shields.io/docker/build/qmcgaw/ddns-updater.svg)](https://hub.docker.com/r/qmcgaw/ddns-updater)

[![GitHub last commit](https://img.shields.io/github/last-commit/qdm12/ddns-updater.svg)](https://github.com/qdm12/ddns-updater/issues)
[![GitHub commit activity](https://img.shields.io/github/commit-activity/y/qdm12/ddns-updater.svg)](https://github.com/qdm12/ddns-updater/issues)
[![GitHub issues](https://img.shields.io/github/issues/qdm12/ddns-updater.svg)](https://github.com/qdm12/ddns-updater/issues)

[![Docker Pulls](https://img.shields.io/docker/pulls/qmcgaw/ddns-updater.svg)](https://hub.docker.com/r/qmcgaw/ddns-updater)
[![Docker Stars](https://img.shields.io/docker/stars/qmcgaw/ddns-updater.svg)](https://hub.docker.com/r/qmcgaw/ddns-updater)
[![Docker Automated](https://img.shields.io/docker/automated/qmcgaw/ddns-updater.svg)](https://hub.docker.com/r/qmcgaw/ddns-updater)

[![Image size](https://images.microbadger.com/badges/image/qmcgaw/ddns-updater.svg)](https://microbadger.com/images/qmcgaw/ddns-updater)
[![Image version](https://images.microbadger.com/badges/version/qmcgaw/ddns-updater.svg)](https://microbadger.com/images/qmcgaw/ddns-updater)

| Image size | RAM usage | CPU usage |
| --- | --- | --- |
| 21.4MB | 13MB | Very low |

## Features

- Updates periodically A records for different DNS providers: Namecheap, GoDaddy, Dreamhost, DuckDNS (ask for more)
- Web User interface

![Web UI](https://raw.githubusercontent.com/qdm12/ddns-updater/master/readme/webui.png)

- Lightweight based on Go and **Alpine 3.9** with Sqlite and Ca-Certificates packages
- Persistence with a sqlite database to store old IP addresses and previous update status

## Setup

1. To setup your domains initially, see the [Domain set up](#domain-set-up) section.
1. Create a *config.json* file owned by user `1000` with read permission

    ```sh
    touch config.json && chown 1000 config.json && chmod 400 config.json
    ```

1. Modify the *config.json* file similarly to:

    ```json
    {
        "settings": [
            {
                "provider": "namecheap",
                "domain": "example.com",
                "host": "@",
                "ip_method": "provider",
                "delay": 86400,
                "password": "e5322165c1d74692bfa6d807100c0310"
            },
            {
                "provider": "duckdns",
                "domain": "example.duckdns.org",
                "ip_method": "provider",
                "token": "00000000-0000-0000-0000-000000000000"
            },
            {
                "provider": "godaddy",
                "domain": "example.org",
                "host": "subdomain",
                "ip_method": "duckduckgo",
                "key": "aaaaaaaaaaaaaaaa",
                "secret": "aaaaaaaaaaaaaaaa"
            },
            {
                "provider": "dreamhost",
                "domain": "example.info",
                "ip_method": "opendns",
                "key": "aaaaaaaaaaaaaaaa"
            }
        ]
    }
    ```

    See more information at the [configuration section](#configuration)

1. Use the following command:

    ```bash
    docker run -d -p 8000:8000/tcp -v $(pwd)/config.json:/updater/config.json:ro qmcgaw/ddns-updater
    ```

    - You can also use [docker-compose.yml](https://github.com/qdm12/ddns-updater/blob/master/docker-compose.yml)
    - You can add history persitence with:

        ```bash
        mkdir data && chown 1000 data && chmod -R 700 data
        docker run -d -p 8000:8000/tcp \
        -v $(pwd)/config.json:/updater/config.json:ro \
        -v $(pwd)/data:/updater/data \
        qmcgaw/ddns-updater
        ```

## Configuration

### Record configuration

The record update updates configuration must be done through the *config.json* mentioned [above](#setup).
**Support for record updates configuration through environment variables will be removed in the coming updates.**
The following parameters are available to all DNS hosts providers:

- `"provider"` is the DNS provider and can be:
    - `godaddy`
    - `namecheap`
    - `duckdns`
    - `dreamhost`
- `"domain"` is your domain name
- `"ip_method"` is the method to obtain your public IP address and can be
    - `provider` means the public IP is automatically determined by the DNS provider (**only for DuckDNs and Namecheap**)
    - `duckduckgo` using [https://duckduckgo.com/?q=ip](https://duckduckgo.com/?q=ip)
    - `opendns` using [https://diagnostic.opendns.com/myip](https://diagnostic.opendns.com/myip)
- `"delay"` is an **optional** integer delay in seconds between each update. It defaults to the `DELAY` environment variable which itself defaults to 5 minutes.

Each DNS provider has a specific set of extra required parameters as follows:

- DuckDNS:
    - `"token"`
- GoDaddy:
    - `"host"` is your host and can be a subdomain, `@` or `*` generally
    - `"key"`
    - `"secret"`
- Namecheap:
    - `"host"` is your host and can be a subdomain, `@` or `*` generally
    - `"password"`
- Dreamhost:
    - `"key"`

### Environment variables

| Environment variable | Default | Description |
| --- | --- | --- |
| `DELAY` | `300` | Delay between updates in seconds |
| `ROOTURL` | `/` | URL path to append to all paths to the webUI (i.e. `/ddns` for accessing `https://example.com/ddns` through a proxy) |
| `LISTENINGPORT` | `8000` | Internal TCP listening port for the web UI |
| `LOGGING` | `json` | Format of logging, `json` or `human` |
| `NODEID` | `0` | Node ID (for distributed systems), can be any integer |

### Host firewall

This container needs the following ports:

- TCP 443 outbound for outbound HTTPS
- UDP 53 outbound for outbound DNS resolution
- TCP 8000 inbound (or other) for the WebUI

## Domain set up

### Namecheap

[![Namecheap Website](https://github.com/qdm12/ddns-updater/raw/master/readme/namecheap.png)](https://www.namecheap.com)

1. Create a Namecheap account and buy a domain name - *example.com* as an example
1. Login to Namecheap at [https://www.namecheap.com/myaccount/login.aspx](https://www.namecheap.com/myaccount/login.aspx)

For **each domain name** you want to add, replace *example.com* in the following link with your domain name and go to [https://ap.www.namecheap.com/Domains/DomainControlPanel/**example.com**/advancedns](https://ap.www.namecheap.com/Domains/DomainControlPanel/example.com/advancedns)

1. For each host you want to add (if you don't know, create one record with the host set to `*`):
    1. In the *HOST RECORDS* section, click on *ADD NEW RECORD*

        ![https://ap.www.namecheap.com/Domains/DomainControlPanel/mealracle.com/advancedns](https://raw.githubusercontent.com/qdm12/ddns-updater/master/readme/namecheap1.png)

    1. Select the following settings and create the *A + Dynamic DNS Record*:

        ![https://ap.www.namecheap.com/Domains/DomainControlPanel/mealracle.com/advancedns](https://raw.githubusercontent.com/qdm12/ddns-updater/master/readme/namecheap2.png)

1. Scroll down and turn on the switch for *DYNAMIC DNS*

    ![https://ap.www.namecheap.com/Domains/DomainControlPanel/mealracle.com/advancedns](https://raw.githubusercontent.com/qdm12/ddns-updater/master/readme/namecheap3.png)

1. The Dynamic DNS Password will appear, which is `0e4512a9c45a4fe88313bcc2234bf547` in this example.

    ![https://ap.www.namecheap.com/Domains/DomainControlPanel/mealracle.com/advancedns](https://raw.githubusercontent.com/qdm12/ddns-updater/master/readme/namecheap4.png)

***

### GoDaddy

[![GoDaddy Website](https://github.com/qdm12/ddns-updater/raw/master/readme/godaddy.png)](https://godaddy.com)

1. Login to [https://developer.godaddy.com/keys](https://developer.godaddy.com/keys/) with your account credentials.

[![GoDaddy Developer Login](https://github.com/qdm12/ddns-updater/raw/master/readme/godaddy1.gif)](https://developer.godaddy.com/keys)

1. Generate a Test key and secret.

[![GoDaddy Developer Test Key](https://github.com/qdm12/ddns-updater/raw/master/readme/godaddy2.gif)](https://developer.godaddy.com/keys)

1. Generate a **Production** key and secret.

[![GoDaddy Developer Production Key](https://github.com/qdm12/ddns-updater/raw/master/readme/godaddy3.gif)](https://developer.godaddy.com/keys)

Obtain the **key** and **secret** of that production key.

In this example, the key is `dLP4WKz5PdkS_GuUDNigHcLQFpw4CWNwAQ5` and the secret is `GuUFdVFj8nJ1M79RtdwmkZ`.

***

### DuckDNS

[![DuckDNS Website](https://github.com/qdm12/ddns-updater/raw/master/readme/duckdns.png)](https://duckdns.org)

*See [duckdns website](https://duckdns.org)*

### Dreamhost

*Awaiting a contribution*

## Testing

- The automated healthcheck verifies all your records are up to date [using DNS lookups](https://github.com/qdm12/ddns-updater/blob/master/healthcheck/main.go)
- You can check manually at:
  - GoDaddy: https://dcc.godaddy.com/manage/yourdomain.com/dns (replace yourdomain.com)

    [![GoDaddy DNS management](https://github.com/qdm12/ddns-updater/raw/master/readme/godaddydnsmanagement.png)](https://dcc.godaddy.com/manage/)

    You might want to try to change the IP address to another one to see if the update actually occurs.
  - Namecheap: *awaiting contribution*
  - DuckDNS: *awaiting contribution*

## Used in external projects

- [Starttoaster/docker-traefik](https://github.com/Starttoaster/docker-traefik#home-networks-extra-credit-dynamic-dns)

## TODOs

- [ ] Cloudflare DNS registrar
- [ ] Unit tests
- [ ] Read parameters from JSON file
- [ ] Finish readme
- [ ] Other types or records
- [ ] ReactJS frontend
    - [ ] Live update of website
    - [ ] Change settings