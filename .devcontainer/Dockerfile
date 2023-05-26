ARG BCI_IMAGE=registry.suse.com/bci/bci-base:latest

FROM ${BCI_IMAGE} as bci
FROM bci

RUN zypper update -y && \
    zypper && \
    zypper install -y   \
        ca-certificates \
        strongswan      \
        curl            \
        git             \
        gcc             \
        glibc-devel     \
        gzip            \
        tar             \
        autoconf        \
        file            \ 
        curl            \
        wget            \
        pigz            \
        awk             \
        make            \
        zlib-devel      \
        libtool         \
        net-tools    && \
    zypper clean --all 

ARG GO_VERSION=1.17
RUN curl -OL https://golang.org/dl/go${GO_VERSION}.linux-amd64.tar.gz
RUN tar -C /usr/local -xvf go${GO_VERSION}.linux-amd64.tar.gz && \
    rm go${GO_VERSION}.linux-amd64.tar.gz

ENV PATH "$PATH:/usr/local/go/bin:/home/user/go/bin"
