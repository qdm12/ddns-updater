ARG BUILDPLATFORM=linux/amd64
ARG ALPINE_VERSION=3.16
ARG GO_VERSION=1.19
ARG XCPUTRANSLATE_VERSION=v0.6.0
ARG GOLANGCI_LINT_VERSION=v1.50.1

FROM --platform=${BUILDPLATFORM} qmcgaw/xcputranslate:${XCPUTRANSLATE_VERSION} AS xcputranslate
FROM --platform=${BUILDPLATFORM} qmcgaw/binpot:golangci-lint-${GOLANGCI_LINT_VERSION} AS golangci-lint

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS base
WORKDIR /tmp/gobuild
ENV CGO_ENABLED=0
RUN apk --update add git g++
COPY --from=xcputranslate /xcputranslate /usr/local/bin/xcputranslate
COPY --from=golangci-lint /bin /go/bin/golangci-lint
# Copy repository code and install Go dependencies
COPY go.mod go.sum ./
RUN go mod download
COPY pkg/ ./pkg/
COPY cmd/ ./cmd/
COPY internal/ ./internal/

FROM --platform=$BUILDPLATFORM base AS test
# Note on the go race detector:
# - we set CGO_ENABLED=1 to have it enabled
# - we installed g++ to support the race detector
ENV CGO_ENABLED=1
ENTRYPOINT go test -race -coverpkg=./... -coverprofile=coverage.txt -covermode=atomic ./...

FROM --platform=$BUILDPLATFORM base AS lint
COPY .golangci.yml ./
RUN golangci-lint run --timeout=10m

FROM --platform=$BUILDPLATFORM base AS build
ARG VERSION=unknown
ARG BUILD_DATE="an unknown date"
ARG COMMIT=unknown
ARG TARGETPLATFORM
RUN GOARCH="$(xcputranslate translate -targetplatform ${TARGETPLATFORM} -field arch)" \
    GOARM="$(xcputranslate translate -targetplatform ${TARGETPLATFORM} -field arm)" \
    go build -trimpath -ldflags="-s -w \
    -X 'main.version=$VERSION' \
    -X 'main.buildDate=$BUILD_DATE' \
    -X 'main.commit=$COMMIT' \
    " -o app cmd/updater/main.go

FROM scratch
EXPOSE 8000
HEALTHCHECK --interval=60s --timeout=5s --start-period=10s --retries=2 CMD ["/updater/app", "healthcheck"]
ARG UID=1000
ARG GID=1000
USER ${UID}:${GID}
ENTRYPOINT ["/updater/app"]
ENV \
    # Core
    CONFIG= \
    PERIOD=5m \
    UPDATE_COOLDOWN_PERIOD=5m \
    PUBLICIP_FETCHERS=all \
    PUBLICIP_HTTP_PROVIDERS=all \
    PUBLICIPV4_HTTP_PROVIDERS=all \
    PUBLICIPV6_HTTP_PROVIDERS=all \
    PUBLICIP_DNS_PROVIDERS=all \
    PUBLICIP_DNS_TIMEOUT=3s \
    HTTP_TIMEOUT=10s \
    DATADIR=/updater/data \

    # Web UI
    LISTENING_PORT=8000 \
    ROOT_URL=/ \

    # Backup
    BACKUP_PERIOD=0 \
    BACKUP_DIRECTORY=/updater/data \

    # Other
    LOG_LEVEL=info \
    LOG_CALLER=hidden \
    SHOUTRRR_ADDRESSES= \
    TZ=
ARG VERSION=unknown
ARG BUILD_DATE="an unknown date"
ARG COMMIT=unknown
LABEL \
    org.opencontainers.image.authors="quentin.mcgaw@gmail.com" \
    org.opencontainers.image.version=$VERSION \
    org.opencontainers.image.created=$BUILD_DATE \
    org.opencontainers.image.revision=$COMMIT \
    org.opencontainers.image.url="https://github.com/qdm12/ddns-updater" \
    org.opencontainers.image.documentation="https://github.com/qdm12/ddns-updater" \
    org.opencontainers.image.source="https://github.com/qdm12/ddns-updater" \
    org.opencontainers.image.title="ddns-updater" \
    org.opencontainers.image.description="Universal DNS updater with WebUI"
COPY --from=build --chown=${UID}:${GID} /tmp/gobuild/app /updater/app
