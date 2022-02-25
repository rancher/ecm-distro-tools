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
    __cmd_check "${DOCKER}"
}

has_git() {
    GIT="$(command -v git)"
    __cmd_check "${GIT}"
}

has_jq() {
    JQ="$(command -v jq)"
    __cmd_check "${JQ}"
}

has_gh() {
    GH="$(command -v gh)"
    __cmd_check "${GH}"
}

has_etcdctl() {
    ETCDCTL="$(command -v etcdctl)"
    __cmd_check "${ETCDCTL}"
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
    TMP_DIR=$(mktemp -d -t ecm.XXXXXXXXXX)
    cleanup() {
        code=$?
        set +e
        trap - EXIT
        rm -rf "${TMP_DIR}"
        exit $code
    }
    trap cleanup INT EXIT

    export TMP_DIR
}

# date functions
day_ago_unix() {
    echo $(($(date +%s) - 86400))
}

week_ago_unix() {
    echo $(($(date +%s) - 604800))
}

month_ago_unix() {
    echo $(($(date +%s) - 2592000))
}

year_ago_unix() {
    echo $(($(date +%s) - 31557600))
}

# colorized output

__RED='\033[0;31m'
__GREEN='\033[0;32m'
__YELLOW='\033[1;33m'
__NC='\033[0m'

print_red() {
    printf "${__RED}%b${__NC}" "$1"
}

print_green() {
    printf "${__GREEN}%b${__NC}" "$1"
}

print_yellow() {
    printf "${__YELLOW}%b${__NC}" "$1"
}
