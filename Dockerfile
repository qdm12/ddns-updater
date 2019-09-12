ARG BASE_IMAGE_BUILDER=golang
ARG BASE_IMAGE=alpine
ARG ALPINE_VERSION=3.10
ARG GO_VERSION=1.13

FROM ${BASE_IMAGE_BUILDER}:${GO_VERSION}-alpine${ALPINE_VERSION} AS builder
ARG GOARCH=amd64
ARG GOARM=
RUN apk --update add git build-base
WORKDIR /tmp/gobuild
COPY go.mod go.sum ./
RUN go mod download 2>&1
COPY pkg/ ./pkg/
COPY main.go .
RUN go test -v ./...
RUN CGO_ENABLED=1 GOOS=linux GOARCH=${GOARCH} GOARM=${GOARM} go build -a -installsuffix cgo -ldflags="-s -w" -o app

FROM ${BASE_IMAGE}:${ALPINE_VERSION} AS final
ARG BUILD_DATE
ARG VCS_REF
LABEL \
    org.opencontainers.image.authors="quentin.mcgaw@gmail.com" \
    org.opencontainers.image.created=$BUILD_DATE \
    org.opencontainers.image.version="" \
    org.opencontainers.image.revision=$VCS_REF \
    org.opencontainers.image.url="https://github.com/qdm12/ddns-updater" \
    org.opencontainers.image.documentation="https://github.com/qdm12/ddns-updater" \
    org.opencontainers.image.source="https://github.com/qdm12/ddns-updater" \
    org.opencontainers.image.title="ddns-updater" \
    org.opencontainers.image.description="Universal DNS updater with WebUI. Works with Namecheap, Cloudflare, GoDaddy, DuckDns, Dreamhost and NoIP" \
    image-size="23.5MB" \
    ram-usage="13MB" \
    cpu-usage="Very Low"
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
ENV DELAY= \
    ROOTURL= \
    LISTENINGPORT= \
    LOGGING= \
    LOGLEVEL= \
    NODEID=
COPY --from=builder --chown=1000 /tmp/gobuild/app /updater/app
COPY --chown=1000 ui/* /updater/ui/
