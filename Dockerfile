FROM rancher/hardened-build-base:v1.17.5b7 AS builder
RUN apk --no-cache add \
    curl \
    file \
    git \
    github-cli \
    gcc \
    bsd-compat-headers \
    py-pip \
    pigz

COPY cmd/gen-release-notes/bin/gen-release-notes /usr/local/bin
COPY cmd/backport/bin/backport /usr/local/bin
COPY cmd/backport/bin/standup /usr/local/bin
COPY bin/. /usr/local/bin
