FROM alpine:3.18.0

# Set the Go version
ARG go_version=1.20

# Install dependencies
RUN apk update

RUN apk add --no-cache git docker docker-compose go make openrc
RUN rc-update add docker boot

# Set Go environment variables
ENV GOROOT=/usr/lib/go \
    GOPATH=/go \
    PATH=/go/bin:/usr/lib/go/bin:$PATH
