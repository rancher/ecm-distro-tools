
# https://github.com/rancher/shell/blob/master/package/Dockerfile#L23-L31
# Needed to speed up the process of building

ARG BCI_IMAGE=registry.suse.com/bci/bci-base:latest
ARG GO_IMAGE=rancher/hardened-build-base:v1.26.1b1
FROM ${BCI_IMAGE} AS bci

# Builder and xx only need to support the host architecture.
FROM --platform=$BUILDPLATFORM rancher/mirrored-tonistiigi-xx:1.6.1 AS xx
FROM --platform=$BUILDPLATFORM ${GO_IMAGE} AS builder

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
ARG TARGETPLATFORM TARGETARCH TARGETOS
ENV ARCH=${TARGETARCH} \
    OS=${TARGETOS}
#RUN mkdir -p /run/lock

RUN xx-go --wrap

RUN OSs=${OS} ARCHS=${ARCH} make all

SHELL ["/bin/bash", "-o", "pipefail", "-c"]

ENV ETCD_VERSION=v3.6.10
RUN if [ "${ARCH}" = "amd64" ] || [ "${ARCH}" = "arm64" ]; then \
        if [ "${ARCH}" = "amd64" ]; then \
            ETCD_SHA256="ed579fafab5701e3aaa95509969e7bc74776a4ae5269d32e3928408b406456ec"; \
        else \
            ETCD_SHA256="e40dc34b7a512b99c651f7700c1014e9b76e02415e2e24365b248966b173524c"; \
        fi; \
        curl -fsSL "https://github.com/etcd-io/etcd/releases/download/${ETCD_VERSION}/etcd-${ETCD_VERSION}-linux-${ARCH}.tar.gz" -o /tmp/etcd.tar.gz && \
        echo "${ETCD_SHA256}  /tmp/etcd.tar.gz" | sha256sum -c - && \
        mkdir -p /tmp/etcd-download && \
        tar xzvf /tmp/etcd.tar.gz -C /tmp/etcd-download --strip-components=1 && \
        rm -f /tmp/etcd.tar.gz && \
        cp /tmp/etcd-download/etcdctl /usr/local/bin; \
    fi

ENV GH_VERSION=v2.89.0
RUN if [ "${ARCH}" = "amd64" ] || [ "${ARCH}" = "arm64" ]; then \
        if [ "${ARCH}" = "amd64" ]; then \
            GH_SHA256="d0422caade520530e76c1c558da47daebaa8e1203d6b7ff10ad7d6faba3490d8"; \
        else \
            GH_SHA256="9e64a623dfc242990aa5d9b3f507111149c4282f66b68eaad1dc79eeb13b9ce5"; \
        fi; \
        curl -fsSL "https://github.com/cli/cli/releases/download/${GH_VERSION}/gh_${GH_VERSION#v}_linux_${ARCH}.tar.gz" -o /tmp/gh.tar.gz && \
        echo "${GH_SHA256}  /tmp/gh.tar.gz" | sha256sum -c - && \
        mkdir -p /tmp/gh-download && \
        tar xzvf /tmp/gh.tar.gz -C /tmp/gh-download --strip-components=1 && \
        rm -f /tmp/gh.tar.gz && \
        cp /tmp/gh-download/bin/gh /usr/local/bin; \
    fi
ENV YQ_VERSION=v4.52.5
RUN if [ "${ARCH}" = "amd64" ] || [ "${ARCH}" = "arm64" ]; then \
        if [ "${ARCH}" = "amd64" ]; then \
            YQ_SHA256="c529c33e6b545d95e39445c37f673e31ca110c3ca9310b47ccea78f9190b061e"; \
        else \
            YQ_SHA256="33cf72426fc5f5e7d9b99801a58d34c6fa7e4426dad594569fb972dde5891109"; \
        fi; \
        curl -fsSL "https://github.com/mikefarah/yq/releases/download/${YQ_VERSION}/yq_linux_${ARCH}.tar.gz" -o /tmp/yq.tar.gz && \
        echo "${YQ_SHA256}  /tmp/yq.tar.gz" | sha256sum -c - && \
        mkdir -p /tmp/yq-download && \
        tar xzvf /tmp/yq.tar.gz -C /tmp/yq-download && \
        rm -f /tmp/yq.tar.gz && \
        cp "/tmp/yq-download/yq_linux_${ARCH}" /usr/local/bin/yq; \
    fi
ENV TRIVY_VERSION=v0.69.3
RUN if [ "${ARCH}" = "amd64" ] || [ "${ARCH}" = "arm64" ]; then \
        if [ "${ARCH}" = "amd64" ]; then \
            TRIVY_SHA256="1816b632dfe529869c740c0913e36bd1629cb7688bd5634f4a858c1d57c88b75"; \
            FILENAME="trivy_${TRIVY_VERSION#v}_Linux-64bit.tar.gz"; \
        else \
            TRIVY_SHA256="7e3924a974e912e57b4a99f65ece7931f8079584dae12eb7845024f97087bdfd"; \
            FILENAME="trivy_${TRIVY_VERSION#v}_Linux-ARM64.tar.gz"; \
        fi; \
        curl -fsSL "https://github.com/aquasecurity/trivy/releases/download/${TRIVY_VERSION}/${FILENAME}" -o /tmp/trivy.tar.gz && \
        echo "${TRIVY_SHA256}  /tmp/trivy.tar.gz" | sha256sum -c - && \
        mkdir -p /tmp/trivy-download && \
        tar xzvf /tmp/trivy.tar.gz -C /tmp/trivy-download && \
        rm -f /tmp/trivy.tar.gz && \
        cp /tmp/trivy-download/trivy /usr/local/bin; \
    fi

FROM bci
ARG TARGETPLATFORM TARGETARCH TARGETOS
ENV ARCH=${TARGETARCH} \
    OS=${TARGETOS}
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
COPY --from=builder /ecm-distro-tools/cmd/backport/bin/backport-${OS}-${ARCH} /usr/local/bin/backport
COPY --from=builder /ecm-distro-tools/cmd/gen_release_report/bin/gen_release_report-${OS}-${ARCH} /usr/local/bin/gen_release_report
COPY --from=builder /ecm-distro-tools/cmd/release/bin/release-${OS}-${ARCH} /usr/local/bin/release
COPY --from=builder /ecm-distro-tools/cmd/rke2_release/bin/rke2_release-${OS}-${ARCH} /usr/local/bin/rke2_release
COPY --from=builder /ecm-distro-tools/cmd/semv/bin/semv-${OS}-${ARCH} /usr/local/bin/semv
COPY --from=builder /ecm-distro-tools/cmd/test_coverage/bin/test_coverage-${OS}-${ARCH} /usr/local/bin/test_coverage
COPY --from=builder /ecm-distro-tools/cmd/upstream_go_version/bin/upstream_go_version-${OS}-${ARCH} /usr/local/bin/upstream_go_version
COPY --from=builder /usr/local/bin/etcdctl /usr/local/bin
COPY --from=builder /usr/local/bin/trivy /usr/local/bin
COPY --from=builder /usr/local/bin/gh /usr/local/bin
COPY --from=builder /usr/local/bin/yq /usr/local/bin
COPY bin/. /usr/local/bin
