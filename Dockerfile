FROM golang:1.25.7-alpine3.23 AS builder

ARG HTTP_PROXY
ARG HTTPS_PROXY
ARG ALL_PROXY
ARG http_proxy
ARG https_proxy
ARG all_proxy

RUN sed -i 's|https://dl-cdn.alpinelinux.org/alpine|https://mirrors.aliyun.com/alpine|g' /etc/apk/repositories

RUN apk update && \
    apk add ca-certificates git bash gcc musl-dev

WORKDIR /opt/src
ADD events events
ADD registry registry
ADD *.go go.mod go.sum ./

RUN go test -v ./registry && \
    go build -o /opt/registry-ui *.go


FROM alpine:3.23

ARG HTTP_PROXY
ARG HTTPS_PROXY
ARG ALL_PROXY
ARG http_proxy
ARG https_proxy
ARG all_proxy

RUN sed -i 's|https://dl-cdn.alpinelinux.org/alpine|https://mirrors.aliyun.com/alpine|g' /etc/apk/repositories

WORKDIR /opt
RUN apk add --no-cache ca-certificates tzdata && \
    mkdir /opt/data && \
    chown nobody /opt/data

ADD templates /opt/templates
ADD static /opt/static
ADD config.yml /opt
COPY --from=builder /opt/registry-ui /opt/

USER nobody
ENTRYPOINT ["/opt/registry-ui"]
