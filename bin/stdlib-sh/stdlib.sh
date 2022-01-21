#!/bin/sh

# return error codes
ERR_GEN=1
ERR_DEP=2
ERR_ARG=3

set_debug() {
    set -x
}

unset_debug() {
    set +x
}

# cmd_check checks if the command output
# passed in is empty or not. If it is then
# we print an error containing the missing
# command and exit.
__cmd_check() {
    if [ -z "$1" ]; then 
        echo "error: $0 requires $1"
        exit ${ERR_DEP}
    fi
}

has_curl() {
    CURL="$(command -v curl)"
    __cmd_check "${CURL}"
}

has_docker() {
    DOCKER="$(command -v docker)"
    cmd_check "${DOCKER}"
}

has_git() {
    GIT="$(command -v git)"
    cmd_check "${GIT}"
}

has_jq() {
    JQ="$(command -v jq)"
    __cmd_check "${JQ}"
}

# setup_verify_arch set arch and suffix,
# fatal if architecture not supported.
setup_verify_arch() {
    if [ -z "$ARCH" ]; then
        ARCH=$(uname -m)
    fi
    case $ARCH in
        amd64)
            ARCH=amd64
            SUFFIX=
            ;;
        x86_64)
            ARCH=amd64
            SUFFIX=
            ;;
        arm64)
            ARCH=arm64
            SUFFIX=-${ARCH}
            ;;
        aarch64)
            ARCH=arm64
            SUFFIX=-${ARCH}
            ;;
        arm*)
            ARCH=arm
            SUFFIX=-${ARCH}hf
            ;;
        *)
            fatal "Unsupported architecture $ARCH"
    esac
}

# setup_tmp create temporary directory 
# and cleanup when done.
setup_tmp() {
    TMP_DIR=$(mktemp -d -t k3s-install.XXXXXXXXXX)
    TMP_HASH=${TMP_DIR}/k3s.hash
    TMP_BIN=${TMP_DIR}/k3s.bin
    cleanup() {
        code=$?
        set +e
        trap - EXIT
        rm -rf "${TMP_DIR}"
        exit $code
    }
    trap cleanup INT EXIT
}