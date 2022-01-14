#!/bin/sh

. ./stdlib.sh

# info logs the given argument at info log level.
info() {
    echo "[INFO] " "$@"
}

# warn logs the given argument at warn log level.
warn() {
    echo "[WARN] " "$@" >&2
}

# fatal logs the given argument at fatal log level.
fatal() {
    echo "[ERROR] " "$@" >&2

    exit 1
}

__timestamp(){
    date "+%Y%m%dT%H%M%S"
}

__log() {
    has_jq

    log_level="$1"
    msg="$2"

    echo '{}' | jq \
        --monochrome-output \
        --compact-output \
        --raw-output \
        --arg timestamp "$(__timestamp)" \
        --arg log_level "$log_level" \
        --arg msg "$msg" \
        '.timestamp=$timestamp|.log_level=$log_level|.msg=$msg'
}
.
info_s() {
    __log "INFO" "$@"
}
.
warn_s() {
    __log "WARN" "$@"
}

fatal_s() {
    __log "FATAL" "$@"

    exit 1
}
