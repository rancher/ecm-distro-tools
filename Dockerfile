
# https://github.com/rancher/shell/blob/master/package/Dockerfile#L23-L31
# Needed to speed up the process of building

ARG BCI_IMAGE=registry.suse.com/bci/bci-base:latest
ARG GO_IMAGE=rancher/hardened-build-base:v1.22.1b1
FROM ${BCI_IMAGE} as bci

# Builder and xx only need to support the host architecture.
FROM --platform=$BUILDPLATFORM rancher/mirrored-tonistiigi-xx:1.3.0 as xx
FROM --platform=$BUILDPLATFORM ${GO_IMAGE} as builder 

# https://github.com/tonistiigi/xx/?tab=readme-ov-file#xx-apk-xx-apt-xx-apt-get---installing-packages-for-target-architecture
RUN apk --no-cache add \
    curl               \
    wget               \
    file               \
    git                \
    github-cli         \
    gcc                \
    bsd-compat-headers \
    py-pip             \
    pigz               \
    tar                \
    yq
COPY . /ecm-distro-tools
WORKDIR /ecm-distro-tools

COPY --from=xx / /

# From this point onwards, although everything will be executed at the
# host architecture, it will fork and run separately for each target
# arch/platform.
ARG TARGETPLATFORM TARGETARCH
#RUN mkdir -p /run/lock

RUN xx-go --wrap

RUN make all
ARG ETCD_VERSION=v3.5.7
ARG GH_VERSION=2.23.0
ARG YQ_VERSION=v4.30.8
RUN mkdir -p /tmp/etcd-download-test                                                                                                                                  && \
    curl -L https://github.com/etcd-io/etcd/releases/download/${ETCD_VERSION}/etcd-${ETCD_VERSION}-linux-amd64.tar.gz -o /tmp/etcd-${ETCD_VERSION}-linux-amd64.tar.gz && \
    tar xzvf /tmp/etcd-${ETCD_VERSION}-linux-amd64.tar.gz -C /tmp/etcd-download-test --strip-components=1                                                             && \
    rm -f /tmp/etcd-${ETCD_VERSION}-linux-amd64.tar.gz                                                                                                                && \
    cp /tmp/etcd-download-test/etcdctl /usr/local/bin
RUN wget https://github.com/aquasecurity/trivy/releases/download/v0.37.3/trivy_0.37.3_Linux-64bit.tar.gz && \
    tar -zxvf trivy_0.37.3_Linux-64bit.tar.gz                                                            && \
    mv trivy /usr/local/bin
RUN wget https://github.com/cli/cli/releases/download/v${GH_VERSION}/gh_${GH_VERSION}_linux_amd64.tar.gz && \
    tar -zxvf gh_${GH_VERSION}_linux_amd64.tar.gz -C /usr/local/bin && \ 
    mv /usr/local/bin/gh_${GH_VERSION}_linux_amd64/bin/gh /usr/local/bin/gh
RUN wget https://github.com/mikefarah/yq/releases/download/${YQ_VERSION}/yq_linux_amd64.tar.gz && \
    tar -zxvf yq_linux_amd64.tar.gz -C /usr/local/bin && \
    mv /usr/local/bin/yq_linux_amd64 /usr/local/bin/yq

FROM bci
RUN zypper update -y && \
    zypper && \
    zypper install -y   \
        ca-certificates \
        strongswan      \ 
        git             \ 
        tar             \
        file            \ 
        curl            \
        wget            \
        pigz            \
        awk             \
        net-tools    && \
    zypper clean --all
COPY --from=builder /ecm-distro-tools/cmd/backport/bin/backport-linux-amd64 /usr/local/bin/backport
COPY --from=builder /ecm-distro-tools/cmd/rpm/bin/rpm-linux-amd64 /usr/local/bin/rpm
COPY --from=builder /ecm-distro-tools/cmd/gen_release_report/bin/gen_release_report-linux-amd64 /usr/local/bin/gen_release_report
COPY --from=builder /ecm-distro-tools/cmd/release/bin/release-linux-amd64 /usr/local/bin/release
COPY --from=builder /ecm-distro-tools/cmd/rancher_release/bin/rancher_release-linux-amd64 /usr/local/bin/rancher_release
COPY --from=builder /ecm-distro-tools/cmd/rke2_release/bin/rke2_release-linux-amd64 /usr/local/bin/rke2_release
COPY --from=builder /ecm-distro-tools/cmd/semv/bin/semv-linux-amd64 /usr/local/bin/semv
COPY --from=builder /ecm-distro-tools/cmd/test_coverage/bin/test_coverage-linux-amd64 /usr/local/bin/test_coverage
COPY --from=builder /ecm-distro-tools/cmd/upstream_go_version/bin/upstream_go_version-linux-amd64 /usr/local/bin/upstream_go_version
COPY --from=builder /usr/local/bin/etcdctl /usr/local/bin
COPY --from=builder /usr/local/bin/trivy /usr/local/bin
COPY --from=builder /usr/local/bin/gh /usr/local/bin
COPY --from=builder /usr/local/bin/yq /usr/local/bin
COPY bin/. /usr/local/bin
