FROM alpine:3.18.0

# Set the Go version
ARG go_version=1.20

# Install dependencies
RUN apk update

RUN apk add --no-cache git docker docker-compose
RUN apk add --no-cache go make
# RUN wget https://golang.org/dl/go1.${go_version}.linux-amd64.tar.gz && \
#     tar -C /usr/local -xzf go1.17.2.linux-amd64.tar.gz && \
#     export PATH=$PATH:/usr/local/go/bin

# Set Go environment variables
ENV GOROOT=/usr/lib/go \
    GOPATH=/go \
    PATH=/go/bin:/usr/lib/go/bin:$PATH
