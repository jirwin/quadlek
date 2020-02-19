FROM golang:1.13-alpine as builder

RUN apk add --update ca-certificates \
    && apk add curl git coreutils \
    && rm /var/cache/apk/*

ENV APP_PATH=/quadlek
RUN mkdir -p $APP_PATH
ADD . $APP_PATH
WORKDIR $APP_PATH
RUN go build -mod=vendor -o /build/quadlekBot ./cmd/quadlek


FROM alpine

RUN apk add --no-cache ca-certificates

COPY --from=builder /build/quadlekBot /

ENTRYPOINT ["/quadlekBot"]
