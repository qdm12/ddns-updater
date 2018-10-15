FROM alpine:3.8 as alpine
RUN apk --update --no-cache --progress add ca-certificates

FROM golang:alpine AS builder
RUN apk --update add git build-base upx
WORKDIR /go/src/healthcheck
COPY healthcheck/*.go ./
RUN go get -v ./... && \
    go test -v && \
    CGO_ENABLED=0 GOOS=linux go build -a -ldflags="-s -w" -installsuffix cgo -o healthcheck . && \
    upx -v --best --ultra-brute --overlay=strip healthcheck && \
    upx -t healthcheck
WORKDIR /go/src/ddns-updater
COPY updater/*.go ./
RUN go get -v ./... && \
    go test -v && \
    CGO_ENABLED=0 GOOS=linux go build -a -ldflags="-s -w" -installsuffix cgo -o updater .

FROM scratch
LABEL maintainer="quentin.mcgaw@gmail.com" \
      description="Lightweight scratch based container updating DNS A records periodically for GoDaddy, Namecheap and DuckDNS" \
      download="???MB" \
      size="???MB" \
      ram="???MB" \
      cpu_usage="Very low" \
      github="https://github.com/qdm12/ddns-updater"
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/src/ddns-updater/updater /updater
COPY --from=builder /go/src/healthcheck/healthcheck /healthcheck
EXPOSE 80
ENTRYPOINT ["/updater"]
HEALTHCHECK --interval=300s --timeout=5s --start-period=5s --retries=1 CMD [ "/healthcheck" ]
COPY updater/index.html /index.html
