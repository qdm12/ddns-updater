# Lightweight universal DDNS Updater with Docker and web UI

*Light container updating DNS A records periodically for GoDaddy, Namecheap, Cloudflare, Dreamhost, NoIP, DNSPod, Infomaniak, ddnss.de and DuckDNS*

[![DDNS Updater by Quentin McGaw](https://github.com/qdm12/ddns-updater/raw/master/readme/title.png)](https://hub.docker.com/r/qmcgaw/ddns-updater)

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

- Updates periodically A records for different DNS providers: Namecheap, GoDaddy, Cloudflare, NoIP, Dreamhost, DuckDNS, DNSPod and Infomaniak (ask for more)
- Web User interface

![Web UI](https://raw.githubusercontent.com/qdm12/ddns-updater/master/readme/webui.png)

- 12.3MB Docker image based on a Go static binary in a Scratch Docker image with ca-certificates and timezone data
- Persistence with a JSON file *updates.json* to store old IP addresses with change times for each record
- Docker healthcheck verifying the DNS resolution of your domains
- Highly configurable
- Sends notifications to your Android phone, see the [**Gotify**](#Gotify) section (it's free, open source and self hosted 🆒)
- Compatible with `amd64`, `386`, `arm64`, `arm32v7` (Raspberry Pis) CPU architectures.

## Setup

1. To setup your domains initially, see the [Domain set up](#domain-set-up) section.
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

    See more information in the [configuration section](#configuration)

1. Use the following command:

    ```bash
    docker run -d -p 8000:8000/tcp -v "$(pwd)"/data:/updater/data qmcgaw/ddns-updater
    ```

    You can also use [docker-compose.yml](https://github.com/qdm12/ddns-updater/blob/master/docker-compose.yml) with:

    ```sh
    docker-compose up -d
    ```

1. You can update the image with `docker pull qmcgaw/ddns-updater`. Other [Docker image tags are available](https://hub.docker.com/repository/docker/qmcgaw/ddns-updater/tags).

## Configuration

Start by having the following content in *config.json*:

```json
{
    "settings": [
        {
            "provider": "",
            "domain": "",
            "ip_method": "",
        },
        {
            "provider": "",
            "domain": "",
            "ip_method": "",
        }
    ]
}
```

The following parameters are to be added in *config.json*

For all record update configuration, you need the following:

- `"provider"` is the DNS provider and can be `"godaddy"`, `"namecheap"`, `"duckdns"`, `"dreamhost"`, `"cloudflare"`, `"noip"`, `"dnspod"` or `"ddnss"`
- `"domain"`
- `"ip_method"` is the method to obtain your public IP address and can be:
  - `"provider"` means the public IP is automatically determined by the DNS provider (**only for DuckDNs, Namecheap, Infomaniak and NoIP**), most reliable.
  - `"opendns"` using [https://diagnostic.opendns.com/myip](https://diagnostic.opendns.com/myip) (reliable)
  - `"ifconfig"` using [https://ifconfig.io/ip](https://ifconfig.io/ip) (may be rate limited)
  - `"ipinfo"` using [https://ipinfo.io/ip](https://ipinfo.io/ip) (may be rate limited)
  - `"ipify"` using [https://api.ipify.org](https://api.ipify.org) (may be rate limited)
  - `"ipify6"` using [https://api6.ipify.org](https://api.ipify.org) for IPv6 only (may be rate limited)
  - `"ddnss"` using [https://ddnss.de/meineip.php](https://ddnss.de/meineip.php)
  - `"ddnss4"` using [https://ip4.ddnss.de/meineip.php](https://ip4.ddnss.de/meineip.php) for IPv4 only
  - `"ddnss6"` using [https://ip6.ddnss.de/meineip.php](https://ip6.ddnss.de/meineip.php) for IPv6 only
  - `"cycle"` to cycle between each external methods, in order to avoid being rate limited
  - You can also specify an HTTPS URL to obtain your public IP address (i.e. `"ip_method": "https://ipinfo.io/ip"`)

You can optionnally add the parameters:

- `"delay"` is the delay in seconds between each update. It defaults to the `DELAY` environment variable value.
- `"no_dns_lookup"` can be `true` or `false` and allows, if `true`, to prevent the periodic Docker healthcheck from running a DNS lookup on your domain.

For each DNS provider exist some specific parameters you need to add, as described below:

Namecheap:

- `"host"` is your host and can be a subdomain, `"@"` or `"*"` generally
- `"password"`

Cloudflare:

- `"zone_identifier"`
- `"identifier"`
- `"host"` is your host and can be a subdomain, `"@"` or `"*"` generally
- `"ttl"` integer value for record TTL in seconds (specify 1 for automatic)
- One of the following:
    - Email `"email"` and key `"key"`
    - User service key `"user_service_key"`
    - API Token `"token"`, configured with DNS edit permissions for your DNS name's zone.
- *Optionally*, `"proxied"` can be `true` or `false` to use the proxy services of Cloudflare

GoDaddy:

- `"host"` is your host and can be a subdomain, `"@"` or `"*"` generally
- `"key"`
- `"secret"`

DuckDNS:

- `"token"`

Dreamhost:

- `"key"`

NoIP:

- `"host"` is your host and can be a subdomain or `"@"`
- `"username"`
- `"password"`

DNSPOD:

- `"host"` is your host and can be a subdomain or `"@"`
- `"token"`

Infomaniak:

- `"user"`
- `"password"`
- `"host"` is your host and can be a subdomain or `"@"`
- `"ip_version"` can be `ipv4` (A records) or `ipv6` (AAAA records)

DDNSS.de:

- `"user"`
- `"password"`
- `"host"` is your host and can be a subdomain or `"@"`
- `"ip_version"` can be `ipv4` (A records) or `ipv6` (AAAA records)

### Environment variables

| Environment variable | Default | Description |
| --- | --- | --- |
| `DELAY` | `10m` | Default delay between updates, following [this format](https://golang.org/pkg/time/#ParseDuration) |
| `ROOT_URL` | `/` | URL path to append to all paths to the webUI (i.e. `/ddns` for accessing `https://example.com/ddns` through a proxy) |
| `LISTENING_PORT` | `8000` | Internal TCP listening port for the web UI |
| `LOG_ENCODING` | `console` | Format of logging, `json` or `console` |
| `LOG_LEVEL` | `info` | Level of logging, `info`, `warning` or `error` |
| `NODE_ID` | `0` | Node ID (for distributed systems), can be any integer |
| `HTTP_TIMEOUT` | `10s` | Timeout for all HTTP requests |
| `GOTIFY_URL` |  | (optional) HTTP(s) URL to your Gotify server |
| `GOTIFY_TOKEN` |  | (optional) Token to access your Gotify server |

### Host firewall

If you have a host firewall in place, this container needs the following ports:

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
- You can check manually at:
  - GoDaddy: [https://dcc.godaddy.com/manage/yourdomain.com/dns](https://dcc.godaddy.com/manage/yourdomain.com/dns) (replace yourdomain.com)

    [![GoDaddy DNS management](https://github.com/qdm12/ddns-updater/raw/master/readme/godaddydnsmanagement.png)](https://dcc.godaddy.com/manage/)

    You might want to try to change the IP address to `127.0.0.1` to see if the update actually occurs.

## Development

1. Setup your environment

    <details><summary>Using VSCode and Docker (easier)</summary><p>

    1. Install [Docker](https://docs.docker.com/install/)
       - On Windows, share a drive with Docker Desktop and have the project on that partition
       - On OSX, share your project directory with Docker Desktop
    1. With [Visual Studio Code](https://code.visualstudio.com/download), install the [remote containers extension](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers)
    1. In Visual Studio Code, press on `F1` and select `Remote-Containers: Open Folder in Container...`
    1. Your dev environment is ready to go!... and it's running in a container :+1: So you can discard it and update it easily!

    </p></details>

    <details><summary>Locally</summary><p>

    1. Install [Go](https://golang.org/dl/), [Docker](https://www.docker.com/products/docker-desktop) and [Git](https://git-scm.com/downloads)
    1. Install Go dependencies with

        ```sh
        go mod download
        ```

    1. Install [golangci-lint](https://github.com/golangci/golangci-lint#install)
    1. You might want to use an editor such as [Visual Studio Code](https://code.visualstudio.com/download) with the [Go extension](https://code.visualstudio.com/docs/languages/go). Working settings are already in [.vscode/settings.json](https://github.com/qdm12/ddns-updater/master/.vscode/settings.json).

    </p></details>

1. Commands available:

    ```sh
    # Build the binary
    go build cmd/app/main.go
    # Test the code
    go test ./...
    # Lint the code
    golangci-lint run
    # Build the Docker image
    docker build -t qmcgaw/ddns-updater .
    ```

1. See [Contributing](https://github.com/qdm12/ddns-updater/master/.github/CONTRIBUTING.md) for more information on how to contribute to this repository.

## Used in external projects

- [Starttoaster/docker-traefik](https://github.com/Starttoaster/docker-traefik#home-networks-extra-credit-dynamic-dns)

## TODOs

- [ ] Update dependencies
- [ ] Mockgen instead of mockery
- [ ] Other types or records
- [ ] icon.ico for webpage
- [ ] Record events log
- [ ] Hot reload of config.json
- [ ] Unit tests
- [ ] ReactJS frontend
    - [ ] Live update of website
    - [ ] Change settings
