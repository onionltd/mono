FROM golang:1.14-alpine

RUN apk add --no-cache make gcc libc-dev

ENV GOPROXY=direct

RUN mkdir -p /go/src/github.com/onionltd/mono

COPY . /go/src/github.com/onionltd/mono

WORKDIR /go/src/github.com/onionltd/mono
