ARG ALPINE_VERSION=3.10
ARG GO_VERSION=1.13

FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS builder
RUN apk --update add git g++
WORKDIR /tmp/gobuild
COPY go.mod go.sum ./
RUN go mod download 2>&1
COPY internal/ ./internal/
COPY cmd/updater/main.go .
#RUN go test -v ./...
RUN CGO_ENABLED=1 go build -a -installsuffix cgo -ldflags="-s -w" -o app

FROM alpine:${ALPINE_VERSION}
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
    org.opencontainers.image.description="Universal DNS updater with WebUI. Works with Namecheap, Cloudflare, GoDaddy, DuckDns, Dreamhost and NoIP"
RUN apk add --update sqlite ca-certificates && \
    mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2 && \
    rm -rf /var/cache/apk/* && \
    # Creating empty database file in case nothing is mounted
    mkdir -p /updater/data && \
    touch /updater/data/updates.db && \
    chown -R 1000 /updater && \
    chmod 700 /updater/data && \
    chmod 700 /updater/data/updates.db
EXPOSE 8000
HEALTHCHECK --interval=10s --timeout=5s --start-period=5s --retries=2 CMD ["/updater/app", "healthcheck"]
USER 1000
ENTRYPOINT ["/updater/app"]
ENV DELAY=10m \
    ROOT_URL=/ \
    LISTENING_PORT=8000 \
    LOG_ENCODING=console \
    LOG_LEVEL=info \
    NODE_ID=0 \
    HTTP_TIMEOUT=10s
COPY --from=builder --chown=1000 /tmp/gobuild/app /updater/app
COPY --chown=1000 ui/* /updater/ui/
