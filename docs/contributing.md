# Contributing

## Table of content

1. [Setup](#Setup)
1. [Commands available](#Commands-available)
1. [Guidelines](#Guidelines)

## Setup

### Using VSCode and Docker

That should be easier and better than a local setup, although it might use more memory if you're not on Linux.

1. Install [Docker](https://docs.docker.com/install/)
    - On Windows, share a drive with Docker Desktop and have the project on that partition
    - On OSX, share your project directory with Docker Desktop
1. With [Visual Studio Code](https://code.visualstudio.com/download), install the [remote containers extension](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers)
1. In Visual Studio Code, press on `F1` and select `Remote-Containers: Open Folder in Container...`
1. Your dev environment is ready to go!... and it's running in a container :+1:

### Locally

Install [Go](https://golang.org/dl/), [Docker](https://www.docker.com/products/docker-desktop) and [Git](https://git-scm.com/downloads); then:

```sh
go mod download
```

And finally install [golangci-lint](https://github.com/golangci/golangci-lint#install).

You might want to use an editor such as [Visual Studio Code](https://code.visualstudio.com/download) with the [Go extension](https://code.visualstudio.com/docs/languages/go). Working settings are already in [.vscode/settings.json](../.vscode/settings.json).

## Build and Run

```sh
go build -o app cmd/updater/main.go
./app
```

## Commands available

- Test the code: `go test ./...`
- Lint the code `golangci-lint run`
- Build the Docker image (tests and lint included): `docker build -t qmcgaw/ddns-updater .`
- Run the Docker container: `docker run -it --rm -v /yourpath/data:/updater/data qmcgaw/ddns-updater`

## Guidelines

The Go code is in the Go file [cmd/updater/main.go](../cmd/updater/main.go) and the [internal directory](../internal), you might want to start reading the main.go file.

See the [Contributing document](../.github/CONTRIBUTING.md) for more information on how to contribute to this repository.
