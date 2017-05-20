FROM alpine

ADD release/quadlekBot /

ENTRYPOINT ["/quadlekBot"]