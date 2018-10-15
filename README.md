# Lightweight DDNS Updater with Docker and web UI

Lightweight scratch based container updating DNS A records periodically for GoDaddy, Namecheap and DuckDNS.

[![DDNS Updater by Quentin McGaw](https://github.com/qdm12/ddns-updater/raw/master/readme/title.png)](https://hub.docker/qmcgaw/ddns-updater)

[![Build Status](https://travis-ci.org/qdm12/ddns-updater.svg?branch=master)](https://travis-ci.org/qdm12/ddns-updater)
[![Docker Build Status](https://img.shields.io/docker/build/qmcgaw/ddns-updater.svg)](https://hub.docker.com/r/qmcgaw/ddns-updater)

[![GitHub last commit](https://img.shields.io/github/last-commit/qdm12/ddns-updater.svg)](https://github.com/qdm12/ddns-updater/issues)
[![GitHub commit activity](https://img.shields.io/github/commit-activity/y/qdm12/ddns-updater.svg)](https://github.com/qdm12/ddns-updater/issues)
[![GitHub issues](https://img.shields.io/github/issues/qdm12/ddns-updater.svg)](https://github.com/qdm12/ddns-updater/issues)

[![Docker Pulls](https://img.shields.io/docker/pulls/qmcgaw/ddns-updater.svg)](https://hub.docker.com/r/qmcgaw/ddns-updater)
[![Docker Stars](https://img.shields.io/docker/stars/qmcgaw/ddns-updater.svg)](https://hub.docker.com/r/qmcgaw/ddns-updater)
[![Docker Automated](https://img.shields.io/docker/automated/qmcgaw/ddns-updater.svg)](https://hub.docker.com/r/qmcgaw/ddns-updater)

[![](https://images.microbadger.com/badges/image/qmcgaw/ddns-updater.svg)](https://microbadger.com/images/qmcgaw/ddns-updater)
[![](https://images.microbadger.com/badges/version/qmcgaw/ddns-updater.svg)](https://microbadger.com/images/qmcgaw/ddns-updater)

| Download size | Image size | RAM usage | CPU usage |
| --- | --- | --- | --- |
| ???MB | ???MB | ???MB | Very low |

## Features

- Updates periodically A records for different DNS providers: Namecheap, GoDaddy, DuckDNS (ask for more)
- Integrated Docker healthcheck program
- Web User interface

![Web UI](https://raw.githubusercontent.com/qdm12/ddns-updater/master/readme/webui.png)

- Very lightweight
    - Scratch with two Golang binaries
        - Healthcheck (UPX-compressed)
        - Main program
    - Ca-Certificates
- Emojis :+1:

## Setup

To setup your domains initially, see the [Domain set up](#domain-set-up) section.

You can then run this container with either Docker:

```bash
docker run -d -p 80:80 -e RECORD1=example.com,@,namecheap,0e4512a9c45a4fe88313bcc2234bf547 qmcgaw/ddns-updater
```

or with [Docker Compose](https://raw.githubusercontent.com/qdm12/ddns-updater/master/docker-compose.yml):

```yml
version: '3'
services:
  ddns-updater:
    image: qmcgaw/ddns-updater
    container_name: ddns-updater
    environment:
      - DELAY=300
      - ROOTURL=
      - LISTENINGPORT=80
      - RECORD1=example.com,@,namecheap,provider,0e4512a9c45a4fe88313bcc2234bf547
      - RECORD2=example.info,@,namecheap,duckduckgo,157fd2a9c45a4fe88313bcc2234bfd58
      - RECORD3=example,www,namecheap,opendns,0e4512a9c45a4fe88313bcc2234bf547      
      - RECORD4=example.info,subdomain,namecheap,222.145.121.59,4a67d2a9c45a4fe88313bcc2234bfd62
      - RECORD5=example.org,*,godaddy,opendns,dLP4WKz5PdkS_GuUDNigHcLQFpw4CWNwAQ5:GuUFdVFj8nJ1M79RtdwmkZ
      - RECORD6=example.app,@,godaddy,duckduckgo,dLP4WKz5PdkS_GuUDNigHcLQFpw4CWNwAQ5:GuUFdVFj8nJ1M79RtdwmkZ
      - RECORD7=example.duckdns.org,@,duckdns,provider,064a0540-864c-4f0f-8bf5-23857452b0c1
      - RECORD8=example2.duckdns.org,@,duckdns,opendns,064a0540-864c-4f0f-8bf5-23857452b0c1
    ports:
      - 8000:80/tcp
    network_mode: bridge
    restart: always
```

```bash
docker-compose up -d
```

The following environement variables are available:

| Environment variable | Default | Description |
| --- | --- | --- |
| `DELAY` | `300` | Delay between updates in seconds |
| `ROOTURL` | `/` | URL path to append to all paths (i.e. `/ddns` for accessing `https://example.com/ddns`) |
| LISTENINGPORT | `80` | Internal TCP listening port for the web UI |
| `RECORDi` | | A record to update in the form `domain_name,host,provider,ip_method,password` |

- The environement variables `RECORD1`, `RECORD2`, etc. are domains to update the IP address for
    - The program reads them, starting at `RECORD1` and will stop as soon as `RECORDn` is not set
    - Each must respect the format `domain_name,host,provider,ip_method,password`
    - The `ip_method` parameter can be:
        - `provider` finds your public IP using your DNS provider (Namecheap or DuckDNS only)
        - `duckduckgo` finds your public IP using [https://duckduckgo.com/?q=ip](https://duckduckgo.com/?q=ip)
        - `opendns` finds your public IP using [https://diagnostic.opendns.com/myip](https://diagnostic.opendns.com/myip)
        - `154.251.67.58` sets your public IP as fixed
- The port mapping `8000:80` is for the web interface
    - [http://localhost:8000](http://localhost:8000) is the main UI list
    - [http://localhost:8000/update](http://localhost:8000/update) is to force the update of your domains

For additional firewall purposes, this container needs the following ports:
- TCP 443 outbound
- UDP 53 outbound
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

### GoDaddy

[![GoDaddy Website](https://github.com/qdm12/ddns-updater/raw/master/readme/godaddy.png)](https://godaddy.com)

1. Login to [https://developer.godaddy.com/keys](https://developer.godaddy.com/keys/) with your account credentials.

[![GoDaddy Developer Login](https://github.com/qdm12/ddns-updater/raw/master/readme/godaddy1.gif)](https://developer.godaddy.com/keys)

2. Generate a Test key and secret.

[![GoDaddy Developer Test Key](https://github.com/qdm12/ddns-updater/raw/master/readme/godaddy2.gif)](https://developer.godaddy.com/keys)

3. Generate a **Production** key and secret.

[![GoDaddy Developer Production Key](https://github.com/qdm12/ddns-updater/raw/master/readme/godaddy3.gif)](https://developer.godaddy.com/keys)

Obtain the **key** and **secret** of that production key.

In this example, the key is `dLP4WKz5PdkS_GuUDNigHcLQFpw4CWNwAQ5` and the secret is `GuUFdVFj8nJ1M79RtdwmkZ`.

### DuckDNS

[![Namecheap Website](https://github.com/qdm12/ddns-updater/raw/master/readme/duckdns.png)](https://duckdns.org)

## Testing

- The automated healthcheck verifies all your records are up to date [using DNS lookups](https://github.com/qdm12/ddns-updater/master/healthcheck/main.go)
- You can check manually at:
  - GoDaddy: https://dcc.godaddy.com/manage/yourdomain.com/dns (replace yourdomain.com)

    [![GoDaddy DNS management](https://github.com/qdm12/ddns-updater/raw/master/readme/godaddydnsmanagement.png)](https://dcc.godaddy.com/manage/)

    You might want to try to change the IP address to another one to see if the update actually occurs.
  - Namecheap:
  - DuckDNS:

## TODOs

- Add favicon.ico
- Finish readme
- Delete old repos
- Live update of website
- Unit tests
- Other types or records
- Better HTML webpage with possibility to change settings