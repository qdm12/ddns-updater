# Lightweight DDNS Updater with Docker and web UI

*Light container updating DNS A records periodically for GoDaddy, Namecheap, Cloudflare, Dreamhost, NoIP, DNSPod and DuckDNS*

**WARNING: Env variables naming changed slightly, see below**

[![DDNS Updater by Quentin McGaw](https://github.com/qdm12/ddns-updater/raw/master/readme/title.png)](https://hub.docker.com/r/qmcgaw/ddns-updater)

[![Build Status](https://travis-ci.org/qdm12/ddns-updater.svg?branch=master)](https://travis-ci.org/qdm12/ddns-updater)
[![Docker Pulls](https://img.shields.io/docker/pulls/qmcgaw/ddns-updater.svg)](https://hub.docker.com/r/qmcgaw/ddns-updater)
[![Docker Stars](https://img.shields.io/docker/stars/qmcgaw/ddns-updater.svg)](https://hub.docker.com/r/qmcgaw/ddns-updater)
[![Image size](https://images.microbadger.com/badges/image/qmcgaw/ddns-updater.svg)](https://microbadger.com/images/qmcgaw/ddns-updater)
[![Image version](https://images.microbadger.com/badges/version/qmcgaw/ddns-updater.svg)](https://microbadger.com/images/qmcgaw/ddns-updater)

[![Join Slack channel](https://img.shields.io/badge/slack-@qdm12-yellow.svg?logo=slack)](https://join.slack.com/t/qdm12/shared_invite/enQtODMwMDQyMTAxMjY1LTU1YjE1MTVhNTBmNTViNzJiZmQwZWRmMDhhZjEyNjVhZGM4YmIxOTMxOTYzN2U0N2U2YjQ2MDk3YmYxN2NiNTc)
[![GitHub last commit](https://img.shields.io/github/last-commit/qdm12/ddns-updater.svg)](https://github.com/qdm12/ddns-updater/issues)
[![GitHub commit activity](https://img.shields.io/github/commit-activity/y/qdm12/ddns-updater.svg)](https://github.com/qdm12/ddns-updater/issues)
[![GitHub issues](https://img.shields.io/github/issues/qdm12/ddns-updater.svg)](https://github.com/qdm12/ddns-updater/issues)

## Features

- Updates periodically A records for different DNS providers: Namecheap, GoDaddy, Cloudflare, NoIP, Dreamhost, DuckDNS (ask for more)
- Web User interface

![Web UI](https://raw.githubusercontent.com/qdm12/ddns-updater/master/readme/webui.png)

- Lightweight based on a Go binary and *Alpine 3.10* with Sqlite and Ca-Certificates packages
- Persistence with a sqlite database to store old IP addresses and previous update status
- Docker healthcheck verifying the DNS resolution of your domains
- Highly configurable
- Sends notifications to your Android phone, see the [**Gotify**](#Gotify) section (it's free, open source and self hosted 🆒)
- Compatible with `amd64`, `386`, `arm64` and `arm32v7` (Raspberry Pis) CPU architectures.

## Setup

1. To setup your domains initially, see the [Domain set up](#domain-set-up) section.
1. Create a directory of your choice, say *data* with a file named **config.json** inside:

    ```sh
    mkdir data
    touch data/config.json
    # Owned by user ID of Docker container (1000)
    chown -R 1000 data
    # all access (for sqlite database)
    chmod 700 data
    # read access only
    chmod 400 data/config.json
    ```

    *(You could change the user ID, for example with `1001`, by running the container with `--user=1001`)*

1. Modify the *data/config.json* file similarly to:

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
            }
        ]
    }
    ```

    See more information at the [configuration section](#configuration)

1. Use the following command:

    ```bash
    docker run -d -p 8000:8000/tcp -v $(pwd)/data:/updater/data qmcgaw/ddns-updater
    ```

    You can also use [docker-compose.yml](https://github.com/qdm12/ddns-updater/blob/master/docker-compose.yml) with:

    ```sh
    docker-compose up -d
    ```

## Configuration

### Record configuration

The record update updates configuration must be done through the *config.json* mentioned [above](#setup).

#### Required parameters for all

- `"provider"` is the DNS provider and can be:
    - `godaddy`
    - `namecheap`
    - `duckdns`
    - `dreamhost`
    - `cloudflare`
    - `noip`
    - `dnspod`
- `"domain"` is your domain name
- `"ip_method"` is the method to obtain your public IP address and can be
    - `provider` means the public IP is automatically determined by the DNS provider (**only for DuckDNs, Namecheap and NoIP**)
    - `duckduckgo` using [https://duckduckgo.com/?q=ip](https://duckduckgo.com/?q=ip)
    - ~`opendns` using [https://diagnostic.opendns.com/myip](https://diagnostic.opendns.com/myip)~ as their https certificate no longer works

Please then refer to your specific DNS host provider in the section below for eventual additional required parameters.

#### Optional parameters for all

- `"delay"` is the delay in seconds between each update. It defaults to the `DELAY` environment variable which itself defaults to 5 minutes.
- `"no_dns_lookup"` is a boolean to prevent the regular Docker healthcheck from running a DNS lookup on your domain. This is useful in some corner cases.

#### Namecheap

- Required:
    - `"host"` is your host and can be a subdomain, `@` or `*` generally
    - `"password"`

#### Cloudflare

- Required:
    - `"zone_identifier"`
    - `"identifier"`
    - `"host"` is your host and can be a subdomain, `@` or `*` generally
    - Either:
        - Email `"email"` and Key `"key"`
        - User service key `"user_service_key"`
- Optional:
    - `"proxied"` is a boolean to use the proxy services of Cloudflare

#### GoDaddy

- Required:
    - `"host"` is your host and can be a subdomain, `@` or `*` generally
    - `"key"`
    - `"secret"`

#### DuckDNS

- Required:
    - `"token"`

#### Dreamhost

- Required:
    - `"key"`

#### NoIP

- Required:
    - `"host"` is your host and can be a subdomain or `@`
    - `"username"` which is your username
    - `"password"`

#### DNSPOD

- Required:
    - `"host"` is your host and can be a subdomain or `@`
    - `"token"`

### Environment variables

| Environment variable | Default | Description |
| --- | --- | --- |
| `DELAY` | `300` | Delay between updates in seconds |
| `ROOT_URL` | `/` | URL path to append to all paths to the webUI (i.e. `/ddns` for accessing `https://example.com/ddns` through a proxy) |
| `LISTENING_PORT` | `8000` | Internal TCP listening port for the web UI |
| `LOG_ENCODING` | `json` | Format of logging, `json` or `human` |
| `LOG_LEVEL` | `info` | Level of logging, `info`, ~`success`~, `warning` or `error` |
| `NODE_ID` | `0` | Node ID (for distributed systems), can be any integer |
| `GOTIFY_URL` |  | HTTP(s) URL to your Gotify server |
| `GOTIFY_TOKEN` |  | Token to access your Gotify server |

### Host firewall

This container needs the following ports:

- TCP 443 outbound for outbound HTTPS
- TCP 80 outbound if you use a local unsecured HTTP connection to your Gotify server
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

### Cloudflare

1. Make sure you have `curl` installed
1. Obtain your API key from Cloudflare website ([see this](https://support.cloudflare.com/hc/en-us/articles/200167836-Where-do-I-find-my-Cloudflare-API-key-))
1. Obtain your zone identifier for your domain name, from the domain's overview page written as *Zone ID*
1. Find your **identifier** in the `id` field with

    ```sh
    ZONEID=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
    EMAIL=example@example.com
    APIKEY=aaaaaaaaaaaaaaaaaa
    curl -X GET "https://api.cloudflare.com/client/v4/zones/$ZONEID/dns_records" \
        -H "X-Auth-Email: $EMAIL" \
        -H "X-Auth-Key: $APIKEY"
    ```

You can now fill in the necessary parameters in *config.json*

Special thanks to @Starttoaster for helping out with the [documentation](https://gist.github.com/Starttoaster/07d568c2a99ad7631dd776688c988326) and testing.

### NoIP

*Awaiting a contribution*

## Gotify

[![https://raw.githubusercontent.com/qdm12/ddns-updater/master/readme/gotify.png](Gotify)](https://gotify.net)

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

- The automated healthcheck verifies all your records are up to date [using DNS lookups](https://github.com/qdm12/ddns-updater/blob/master/healthcheck/main.go)
- You can check manually at:
  - GoDaddy: https://dcc.godaddy.com/manage/yourdomain.com/dns (replace yourdomain.com)

    [![GoDaddy DNS management](https://github.com/qdm12/ddns-updater/raw/master/readme/godaddydnsmanagement.png)](https://dcc.godaddy.com/manage/)

    You might want to try to change the IP address to another one to see if the update actually occurs.
  - Namecheap: *awaiting contribution*
  - DuckDNS: *awaiting contribution*

## Used in external projects

- [Starttoaster/docker-traefik](https://github.com/Starttoaster/docker-traefik#home-networks-extra-credit-dynamic-dns)

## Development

### Using VSCode and Docker

1. Install [Docker](https://docs.docker.com/install/)
    - On Windows, share a drive with Docker Desktop and have the project on that partition
1. With [Visual Studio Code](https://code.visualstudio.com/download), install the [remote containers extension](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers)
1. In Visual Studio Code, press on `F1` and select `Remote-Containers: Open Folder in Container...`
1. Your dev environment is ready to go!... and it's running in a container :+1:

## TODOs

- [ ] icon.ico for webpage
- [ ] Record events log
- [ ] Use internal package instead of pkg
- [ ] Hot reload of config.json
- [ ] Changed from sqlite to bolt or similar
- [ ] Unit tests
- [ ] Other types or records
- [ ] ReactJS frontend
    - [ ] Live update of website
    - [ ] Change settings
