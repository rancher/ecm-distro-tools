ARG BCI_IMAGE=registry.suse.com/bci/bci-base:latest
FROM ${BCI_IMAGE} as bci
FROM bci
RUN zypper update -y && \
    zypper && \
    zypper install -y   \
        ca-certificates \
        strongswan      \ 
        git             \
        go              \
        go-doc          \
        tar             \
        file            \ 
        curl            \
        wget            \
        pigz            \
        awk             \
        make            \
        zlib-devel      \
        net-tools    

ENV PATH "$PATH:/usr/local/go/bin:/home/user/go/bin"
