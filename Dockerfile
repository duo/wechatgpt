FROM golang:1.19-alpine AS builder

WORKDIR /build

COPY ./ .

RUN set -ex \
	&& cd /build \
	&& go build -o wechatgpt

FROM alpine:latest

RUN apk add --no-cache --update --quiet --no-progress ca-certificates

COPY --from=builder /build/wechatgpt /usr/bin/wechatgpt

VOLUME /app
WORKDIR /app

CMD ["/usr/bin/wechatgpt"]
