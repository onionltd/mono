FROM alpine:latest

COPY rootfs /

ARG TOR_RELEASE

RUN apk add --no-cache git libevent-dev openssl-dev \
        gcc make automake ca-certificates autoconf \
        musl-dev coreutils zlib-dev \
    && git clone --branch tor-${TOR_RELEASE} --depth 1 https://git.torproject.org/tor.git /tmp/tor \
    # TODO: verify pgp signature!
    && cd /tmp/tor \
    && ./autogen.sh \
    && ./configure \
        --disable-asciidoc \
        --sysconfdir=/etc \
        --disable-unittests \
    && make \
    && make install \
    && cd .. \
    && rm -rf /tmp/tor

RUN addgroup -S -g 107 tor \
    && adduser -S -G tor -u 104 tor
