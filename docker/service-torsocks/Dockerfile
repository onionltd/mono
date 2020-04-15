FROM localhost/gobuilder:latest AS gobuilder

ARG SERVICE_NAME

RUN make build -C services/${SERVICE_NAME}

FROM alpine:latest

ARG SERVICE_NAME

RUN apk add --no-cache torsocks

COPY rootfs /
COPY --from=gobuilder /go/src/github.com/onionltd/mono/services/${SERVICE_NAME}/${SERVICE_NAME} /

RUN adduser -D user

USER user

ENV SERVICE=${SERVICE_NAME}
ENV TORSOCKS_CONF_FILE=/home/user/.torsocks.conf

ENTRYPOINT ["/entrypoint.sh"]

CMD /$SERVICE
