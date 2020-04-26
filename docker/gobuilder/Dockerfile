FROM golang:1.14-alpine

RUN apk add --no-cache make gcc libc-dev git

RUN mkdir -p /go/src/github.com/onionltd/mono
WORKDIR /go/src/github.com/onionltd/mono

COPY . .
ENV GOPROXY=direct
RUN go mod vendor -v
