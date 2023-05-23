FROM alpine:3.18.0

# Set the Go version
ARG go_version=1.20

# Install dependencies
RUN apk update \
    && apk add --no-cache \
        go${go_version} \
        git \
        docker \
        docker-compose

# Set Go environment variables
ENV GOROOT=/usr/lib/go \
    GOPATH=/go \
    PATH=/go/bin:/usr/lib/go/bin:$PATH
