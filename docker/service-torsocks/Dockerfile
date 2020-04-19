FROM localhost/gobuilder:latest AS gobuilder

ARG SERVICE_NAME

RUN make build -C services/${SERVICE_NAME} && make export -C services/${SERVICE_NAME}

FROM alpine:latest

ARG SERVICE_NAME

RUN apk add --no-cache torsocks

COPY rootfs /

RUN mkdir -p /opt/${SERVICE_NAME}/bin
COPY --from=gobuilder /go/src/github.com/onionltd/mono/services/${SERVICE_NAME}/${SERVICE_NAME} /opt/${SERVICE_NAME}/bin
COPY --from=gobuilder /go/src/github.com/onionltd/mono/services/${SERVICE_NAME}/exported /opt/${SERVICE_NAME}

RUN adduser -D user

USER user

ENV SERVICE=${SERVICE_NAME}
ENV TORSOCKS_CONF_FILE=/home/user/.torsocks.conf

ENTRYPOINT ["/entrypoint.sh"]

CMD /opt/$SERVICE/bin/$SERVICE
