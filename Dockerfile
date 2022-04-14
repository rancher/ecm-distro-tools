ARG UBI_IMAGE=registry.suse.com/bci/bci-base:15.3.17.11.11
ARG GO_IMAGE=rancher/hardened-build-base:v1.17.8b7
FROM ${UBI_IMAGE} as bci
FROM ${GO_IMAGE} as builder
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
COPY . /ecm-distro-tools
WORKDIR /ecm-distro-tools
RUN make all
ARG ETCD_VERSION=v3.5.2
RUN mkdir -p /tmp/etcd-download-test                                                                                                                                  && \
    curl -L https://github.com/etcd-io/etcd/releases/download/${ETCD_VERSION}/etcd-${ETCD_VERSION}-linux-amd64.tar.gz -o /tmp/etcd-${ETCD_VERSION}-linux-amd64.tar.gz && \
    tar xzvf /tmp/etcd-${ETCD_VERSION}-linux-amd64.tar.gz -C /tmp/etcd-download-test --strip-components=1                                                             && \
    rm -f /tmp/etcd-${ETCD_VERSION}-linux-amd64.tar.gz                                                                                                                && \
    cp /tmp/etcd-download-test/etcdctl /usr/local/bin
RUN wget https://github.com/aquasecurity/trivy/releases/download/v0.25.3/trivy_0.25.3_Linux-64bit.tar.gz && \
    tar -zxvf trivy_0.25.3_Linux-64bit.tar.gz                                                            && \
    mv trivy /usr/local/bin

FROM bci
RUN zypper addrepo https://cli.github.com/packages/rpm/gh-cli.repo && \
    zypper update -y && \
    zypper --kgpg-auto-import-keys refresh && \
    zypper install -y   \
        ca-certificates \
        strongswan      \ 
        git             \ 
        gh              \
        file            \ 
        curl            \
        yq              \ 
        pigz            \
        py-pip          \
        net-tools    && \
    zypper clean --all
COPY --from=builder /ecm-distro-tools/cmd/gen-release-notes/bin/gen-release-notes /usr/local/bin
COPY --from=builder /ecm-distro-tools/cmd/backport/bin/backport /usr/local/bin
COPY --from=builder /ecm-distro-tools/cmd/standup/bin/standup /usr/local/bin
COPY --from=builder /usr/local/bin/etcdctl /usr/local/bin
COPY --from=builder /usr/local/bin/trivy /usr/local/bin
COPY bin/. /usr/local/bin
