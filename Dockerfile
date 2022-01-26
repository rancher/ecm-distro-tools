FROM rancher/hardened-build-base:v1.17.5b7 AS builder
RUN apk --no-cache add \
    curl \
    file \
    git \
    gcc \
    bsd-compat-headers \
    py-pip \
    pigz

