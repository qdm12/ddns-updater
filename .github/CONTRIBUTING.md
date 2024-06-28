# Contributing

## Table of content

1. [Submitting a pull request](#submitting-a-pull-request)
1. [Development setup](#development-setup)
1. [Commands available](#commands-available)
1. [Add a new DNS provider](#add-a-new-dns-provider)
1. [License](#license)

## Submitting a pull request

1. [Fork](https://github.com/qdm12/ddns-updater/fork) and clone the repository
1. Create a new branch `git checkout -b my-branch-name`
1. Modify the code
1. Commit your modifications
1. Push to your fork and [submit a pull request](https://github.com/qdm12/ddns-updater/compare)

Additional resources:

- [Using Pull Requests](https://help.github.com/articles/about-pull-requests/)
- [How to Contribute to Open Source](https://opensource.guide/how-to-contribute/)

## Development setup

### Using VSCode and Docker

That should be easier and better than a local setup, although it might use more memory if you're not on Linux.

1. Install [Docker](https://docs.docker.com/install/)
    - On Windows, share a drive with Docker Desktop and have the project on that partition
    - On OSX, share your project directory with Docker Desktop
1. With [Visual Studio Code](https://code.visualstudio.com/download), install the [dev containers extension](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers)
1. In Visual Studio Code, press on `F1` and select `Dev Containers: Open Folder in Container...`
1. Your dev environment is ready to go!... and it's running in a container :+1:

### Locally

Install [Go](https://golang.org/dl/), [Docker](https://www.docker.com/products/docker-desktop) and [Git](https://git-scm.com/downloads); then:

```sh
go mod download
```

And finally install [golangci-lint](https://github.com/golangci/golangci-lint#install).

You might want to use an editor such as [Visual Studio Code](https://code.visualstudio.com/download) with the [Go extension](https://code.visualstudio.com/docs/languages/go).

## Commands available

- Test the code: `go test ./...`
- Lint the code `golangci-lint run`
- Build the program: `go build -o app cmd/ddns-updater/main.go`
- Build the Docker image (tests and lint included): `docker build -t qmcgaw/ddns-updater .`
- Run the Docker container: `docker run -it --rm -v /yourpath/data:/updater/data qmcgaw/ddns-updater`

## Add a new DNS provider

An "example" DNS provider is present in the code, you can simply copy paste it modify it to your needs.
In more detailed steps:

1. Copy the directory [`internal/provider/providers/example`](../internal/provider/providers/example) to `internal/provider/providers/yourprovider` where `yourprovider` is the name of the DNS provider you want to add, in a single word without spaces, dashes or underscores.
1. Modify the `internal/provider/providers/yourprovider/provider.go` file to fit the requirements of your DNS provider. There are many `// TODO` comments you can follow and **need to remove** when done.
1. Add the provider name constant to the `ProviderChoices` function in [`internal/provider/constants/providers.go`](../internal/provider/constants/providers.go). For example:

    ```go
    func ProviderChoices() []models.Provider {
      return []models.Provider{
        // ...
        Example,
        // ...
      }
    }
    ```

1. Add a case for your provider in the `switch` statement in the `New` function in [`internal/provider/provider.go`](../internal/provider/provider.go). For example:

    ```go
    case constants.Example:
      return example.New(data, domain, owner, ipVersion, ipv6Suffix)
    ```

1. Copy the file [`docs/example.md`](../docs/example.md) to `docs/yourprovider.md` and modify it to fit the configuration and domain setup of your DNS provider. There are a few `<!-- ... -->` comments indicating what to change, please **remove them** when done.
1. In the [README.md](../README.md):
    1. Add your provider name to the  list of providers supported `- Your provider`
    1. Add your provider name and link to its document to the second list: `- [Your provider](docs/yourprovider.md)`
1. Make sure to run the actual program (in Docker or directly) and check it updates your DNS records as expected, of course 😉 You can do this by setting a record to `127.0.0.1` manually and then run the updater to see if the update succeeds.
1. Profit 🎉 Don't forget to [open a pull request](https://github.com/qdm12/ddns-updater/compare)

## License

Contributions are [released](https://help.github.com/articles/github-terms-of-service/#6-contributions-under-repository-license) to the public under the [open source license of this project](../LICENSE).
