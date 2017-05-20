FROM lwolf/golang-glide:0.12.3 AS build-env

ENV APP_PATH=/go/src/github.com/jirwin/quadlek
RUN mkdir -p $APP_PATH
ADD . $APP_PATH
WORKDIR $APP_PATH
COPY glide.yaml glide.yaml
COPY glide.lock glide.lock
RUN glide install -v && go build -o /build/quadlekBot

FROM alpine
COPY --from=build-env /build/quadlekBot /quadlekBot
ENTRYPOINT ./quadlekBot