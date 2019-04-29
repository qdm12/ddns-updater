ARG ALPINE_VERSION=3.9
ARG GO_VERSION=1.12.4

FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS builder
ARG BINCOMPRESS
RUN apk --update add git build-base upx
RUN go get -u -v golang.org/x/vgo
WORKDIR /tmp/gobuild
COPY go.mod go.sum ./
RUN go mod download
COPY pkg/ ./pkg/
COPY main.go .
#RUN go test -v
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-s -w" -o app .
RUN [ "${BINCOMPRESS}" == "" ] || (upx -v --best --ultra-brute --overlay=strip app && upx -t app)

FROM alpine:${ALPINE_VERSION} AS final
ARG BUILD_DATE
ARG VCS_REF
LABEL org.label-schema.schema-version="1.0.0-rc1" \
      maintainer="quentin.mcgaw@gmail.com" \
      org.label-schema.build-date=$BUILD_DATE \
      org.label-schema.vcs-ref=$VCS_REF \
      org.label-schema.vcs-url="https://github.com/qdm12/ddns-updater" \
      org.label-schema.url="https://github.com/qdm12/ddns-updater" \
      org.label-schema.vcs-description="Lightweight container updating DNS A records periodically for GoDaddy, Namecheap, Dreamhost and DuckDNS" \
      org.label-schema.vcs-usage="https://github.com/qdm12/ddns-updater/blob/master/README.md#setup" \
      org.label-schema.docker.cmd="docker run -d -p 8000:8000/tcp -e RECORD1=example.com,@,namecheap,provider,0e4512a9c45a4fe88313bcc2234bf547 qmcgaw/ddns-updater" \
      org.label-schema.docker.cmd.devel="docker run -it --rm -p 8000:8000/tcp -e RECORD1=example.com,@,namecheap,provider,0e4512a9c45a4fe88313bcc2234bf547 qmcgaw/ddns-updater" \
      org.label-schema.docker.params="See readme" \
      org.label-schema.version="" \
      image-size="21.4MB" \
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
    RECORD0= \
    LOGGING= \
    NODEID=
COPY --from=builder --chown=1000 /tmp/gobuild/app /updater/app
COPY --chown=1000 ui/* /updater/ui/
