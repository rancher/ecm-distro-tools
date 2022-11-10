ARG BCI_IMAGE=registry.suse.com/bci/bci-base:15.3.17.11.11
ARG GO_IMAGE=rancher/hardened-build-base:v1.17.8b7
FROM ${BCI_IMAGE} as bci
FROM ${GO_IMAGE} as builder
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
RUN cd ./cmd/backport && make LDFLAGS="-linkmode=external"
RUN cd ./cmd/gen_release_notes && make LDFLAGS="-linkmode=external"
RUN cd ./cmd/gen_release_report && make LDFLAGS="-linkmode=external"
RUN cd ./cmd/k3s_release && make LDFLAGS="-linkmode=external"
RUN cd ./cmd/standup && make
RUN go-assert-static.sh \
        ./cmd/backport/bin//backport \
        ./cmd/gen_release_notes/bin//gen_release_notes \
        ./cmd/k3s_release/bin/k3s_release \
        ./cmd/standup/bin/standup
RUN go-assert-boring.sh \
        ./cmd/backport/bin//backport \
        ./cmd/gen_release_notes/bin//gen_release_notes \
        ./cmd/k3s_release/bin/k3s_release
ARG ETCD_VERSION=v3.5.2
ARG GH_VERSION=2.8.0
ARG YQ_VERSION=v4.24.4
RUN mkdir -p /tmp/etcd-download-test                                                                                                                                  && \
    curl -L https://github.com/etcd-io/etcd/releases/download/${ETCD_VERSION}/etcd-${ETCD_VERSION}-linux-amd64.tar.gz -o /tmp/etcd-${ETCD_VERSION}-linux-amd64.tar.gz && \
    tar xzvf /tmp/etcd-${ETCD_VERSION}-linux-amd64.tar.gz -C /tmp/etcd-download-test --strip-components=1                                                             && \
    rm -f /tmp/etcd-${ETCD_VERSION}-linux-amd64.tar.gz                                                                                                                && \
    cp /tmp/etcd-download-test/etcdctl /usr/local/bin
RUN wget https://github.com/aquasecurity/trivy/releases/download/v0.25.3/trivy_0.25.3_Linux-64bit.tar.gz && \
    tar -zxvf trivy_0.25.3_Linux-64bit.tar.gz                                                            && \
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
COPY --from=builder /ecm-distro-tools/cmd/gen_release_notes/bin/gen_release_notes /usr/local/bin
COPY --from=builder /ecm-distro-tools/cmd/gen_release_report/bin/gen_release_report /usr/local/bin
COPY --from=builder /ecm-distro-tools/cmd/k3s_release/bin/k3s_release /usr/local/bin
COPY --from=builder /ecm-distro-tools/cmd/backport/bin/backport /usr/local/bin
COPY --from=builder /ecm-distro-tools/cmd/standup/bin/standup /usr/local/bin
COPY --from=builder /usr/local/bin/etcdctl /usr/local/bin
COPY --from=builder /usr/local/bin/trivy /usr/local/bin
COPY --from=builder /usr/local/bin/gh /usr/local/bin
COPY --from=builder /usr/local/bin/yq /usr/local/bin
COPY bin/. /usr/local/bin
