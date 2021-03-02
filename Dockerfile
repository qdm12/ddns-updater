ARG ALPINE_VERSION=3.13
ARG GO_VERSION=1.15
ARG BUILDPLATFORM=linux/amd64

FROM --platform=$BUILDPLATFORM alpine:${ALPINE_VERSION} AS alpine
RUN apk --update add ca-certificates tzdata
RUN mkdir /tmp/data && \
    chown 1000 /tmp/data && \
    chmod 700 /tmp/data

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS base
ENV CGO_ENABLED=0
RUN apk --update add git
WORKDIR /tmp/gobuild
# Copy repository code and install Go dependencies
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ ./cmd/
COPY internal/ ./internal/

FROM --platform=$BUILDPLATFORM base AS test
ENV CGO_ENABLED=1
# g++ is installed for the -race detector in go test
RUN apk --update add g++

FROM --platform=$BUILDPLATFORM base AS lint
ARG GOLANGCI_LINT_VERSION=v1.33.0
RUN wget -O- -nv https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | \
    sh -s -- -b /usr/local/bin ${GOLANGCI_LINT_VERSION}
COPY .golangci.yml ./
RUN golangci-lint run --timeout=10m

FROM --platform=$BUILDPLATFORM base AS tidy
RUN git init && \
    git config user.email ci@localhost && \
    git config user.name ci && \
    git add -A && git commit -m ci && \
    sed -i '/\/\/ indirect/d' go.mod && \
    go mod tidy && \
    git diff --exit-code -- go.mod

FROM --platform=$BUILDPLATFORM base AS build
COPY --from=qmcgaw/xcputranslate:v0.4.0 /xcputranslate /usr/local/bin/xcputranslate
ARG TARGETPLATFORM
ARG VERSION=unknown
ARG BUILD_DATE="an unknown date"
ARG COMMIT=unknown
RUN GOARCH="$(xcputranslate -targetplatform ${TARGETPLATFORM} -field arch)" \
    GOARM="$(xcputranslate -targetplatform ${TARGETPLATFORM} -field arm)" \
    go build -trimpath -ldflags="-s -w \
    -X 'main.version=$VERSION' \
    -X 'main.buildDate=$BUILD_DATE' \
    -X 'main.commit=$COMMIT' \
    " -o app cmd/updater/main.go

FROM scratch
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
    UPDATE_COOLDOWN_PERIOD=5m \
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
    LOG_LEVEL=info \
    LOG_CALLER=hidden \
    GOTIFY_URL= \
    GOTIFY_TOKEN= \
    TZ=
COPY --from=alpine --chown=1000 /tmp/data /updater/data/
COPY --from=build --chown=1000 /tmp/gobuild/app /updater/app
COPY --chown=1000 ui/* /updater/ui/
