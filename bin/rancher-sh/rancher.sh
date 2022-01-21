#!/bin/sh

. ../stdlib-sh/stdlib.sh

__RANCHER_PATH="${GOPATH}/src/github.com/rancher"
__INVALID_ARG_ERROR="error: update requires an argument"
__IMAGE_BUILD_REPOS=$(find "${__RANCHER_PATH}" \
    -path "*image-build-*"                     \
    -type f -not -path "*image-build-tools/*"  \
    -type f -not -path "*image-build-base/*"   \
    -type f -name "Dockerfile")

# rancher_list_local_repos prints out all of the local
# repos to STDOUT.
rancher_list_local_repos() {
    for i in ${__IMAGE_BUILD_REPOS}; do
        echo "$i" | sed 's/\/Dockerfile//g'
    done
}
