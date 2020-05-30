ARG ALPINE_VERSION=3.11
ARG GO_VERSION=1.14

FROM alpine:${ALPINE_VERSION} AS alpine
RUN apk --update add ca-certificates tzdata

FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS builder
ARG GOLANGCI_LINT_VERSION=v1.27.0
RUN apk --update add git
ENV CGO_ENABLED=0
RUN wget -O- -nv https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s ${GOLANGCI_LINT_VERSION}
WORKDIR /tmp/gobuild
COPY .golangci.yml .
COPY go.mod go.sum ./
RUN go mod download 2>&1
COPY internal/ ./internal/
COPY cmd/updater/main.go .
RUN go test ./...
RUN go build -ldflags="-s -w" -o app
RUN golangci-lint run --timeout=10m

FROM scratch
ARG BUILD_DATE
ARG VCS_REF
ARG VERSION
LABEL \
    org.opencontainers.image.authors="quentin.mcgaw@gmail.com" \
    org.opencontainers.image.created=$BUILD_DATE \
    org.opencontainers.image.version=$VERSION \
    org.opencontainers.image.revision=$VCS_REF \
    org.opencontainers.image.url="https://github.com/qdm12/ddns-updater" \
    org.opencontainers.image.documentation="https://github.com/qdm12/ddns-updater" \
    org.opencontainers.image.source="https://github.com/qdm12/ddns-updater" \
    org.opencontainers.image.title="ddns-updater" \
    org.opencontainers.image.description="Universal DNS updater with WebUI. Works with Namecheap, Cloudflare, GoDaddy, DuckDns, Dreamhost, DNSPod and NoIP"
COPY --from=alpine --chown=1000 /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=alpine --chown=1000 /usr/share/zoneinfo /usr/share/zoneinfo
EXPOSE 8000
HEALTHCHECK --interval=60s --timeout=5s --start-period=10s --retries=2 CMD ["/updater/app", "healthcheck"]
USER 1000
ENTRYPOINT ["/updater/app"]
ENV \
    # Core
    CONFIG= \
    PERIOD=5m \
    IP_METHOD=cycle \
    IPV4_METHOD=cycle \
    IPV6_METHOD=cycle \
    HTTP_TIMEOUT=10s \

    # Web UI
    LISTENING_PORT=8000 \
    ROOT_URL=/ \

    # Backup
    BACKUP_PERIOD=0 \
    BACKUP_DIRECTORY=/updater/data \

    # Other
    LOG_ENCODING=console \
    LOG_LEVEL=info \
    NODE_ID=-1 \
    GOTIFY_URL= \
    GOTIFY_TOKEN=
COPY --from=builder --chown=1000 /tmp/gobuild/app /updater/app
COPY --chown=1000 ui/* /updater/ui/
