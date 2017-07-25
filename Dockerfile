FROM alpine:3.5

ADD https://github.com/emailbob/coreos-update-logger/releases/download/latest-linux/coreos-update-logger /coreos-update-logger

RUN apk add --update ca-certificates && \
    rm -rf /var/cache/apk/* && \
    chmod +x /coreos-update-logger

CMD ["/coreos-update-logger"]