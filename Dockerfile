ARG ALPINE_VERSION=3.8

FROM golang:alpine AS builder
RUN apk --update add git build-base upx
RUN go get -u -v golang.org/x/vgo
WORKDIR /tmp/gobuild

FROM alpine:${ALPINE_VERSION} AS packages
RUN apk --update add ca-certificates

FROM scratch AS final
ARG BUILD_DATE
ARG VCS_REF
LABEL org.label-schema.schema-version="1.0.0-rc1" \
      maintainer="quentin.mcgaw@gmail.com" \
      org.label-schema.build-date=$BUILD_DATE \
      org.label-schema.vcs-ref=$VCS_REF \
      org.label-schema.vcs-url="https://github.com/qdm12/ddns-updater" \
      org.label-schema.url="https://github.com/qdm12/ddns-updater" \
      org.label-schema.vcs-description="Lightweight container updating DNS A records periodically for GoDaddy, Namecheap and DuckDNS" \
      org.label-schema.vcs-usage="https://github.com/qdm12/ddns-updater/blob/master/README.md#setup" \
      org.label-schema.docker.cmd="docker run -d -p 80:80 -e RECORD1=example.com,@,namecheap,0e4512a9c45a4fe88313bcc2234bf547 qmcgaw/ddns-updater" \
      org.label-schema.docker.cmd.devel="docker run -it --rm -p 80:80 -e RECORD1=example.com,@,namecheap,0e4512a9c45a4fe88313bcc2234bf547 qmcgaw/ddns-updater" \
      org.label-schema.docker.params="" \
      org.label-schema.version="" \
      image-size="MB" \
      ram-usage="13MB" \
      cpu-usage="Very Low"
COPY --from=packages /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
EXPOSE 80
HEALTHCHECK --interval=300s --timeout=5s --start-period=5s --retries=1 CMD ["/healthcheck"]
ENTRYPOINT ["/updater"]

FROM builder AS builder-healthcheck
COPY healthcheck/*.go ./
RUN go test -v
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-s -w" -o app .
RUN upx -v --best --ultra-brute --overlay=strip app && upx -t app

FROM builder AS builder-updater
COPY updater/go.mod updater/go.sum ./
RUN go mod download
COPY updater/*.go ./
RUN go test -v
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-s -w" -o app .

FROM final
COPY --from=builder-healthcheck /tmp/gobuild/app /healthcheck
COPY --from=builder-updater /tmp/gobuild/app /updater
COPY updater/index.html /index.html
