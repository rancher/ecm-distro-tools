FROM rancher/hardened-build-base:v1.17.5b7 AS builder
RUN apk --no-cache add \
    curl               \
    file               \
    git                \
    github-cli         \
    gcc                \
    bsd-compat-headers \
    py-pip             \
    pigz               \
    yq

ARG ETCD_VERSION=v3.5.2
RUN mkdir -p /tmp/etcd-download-test                                                                                                                                  && \
    curl -L https://github.com/etcd-io/etcd/releases/download/${ETCD_VERSION}/etcd-${ETCD_VERSION}-linux-amd64.tar.gz -o /tmp/etcd-${ETCD_VERSION}-linux-amd64.tar.gz && \
    tar xzvf /tmp/etcd-${ETCD_VERSION}-linux-amd64.tar.gz -C /tmp/etcd-download-test --strip-components=1                                                             && \
    rm -f /tmp/etcd-${ETCD_VERSION}-linux-amd64.tar.gz                                                                                                                && \
    cp /tmp/etcd-download-test/etcdctl /usr/local/bin

COPY cmd/gen-release-notes/bin/gen-release-notes /usr/local/bin
COPY cmd/backport/bin/backport /usr/local/bin
COPY cmd/standup/bin/standup /usr/local/bin
COPY bin/. /usr/local/bin
