#!/bin/sh

__INVALID_ARG_ERROR="error: update requires an argument"

# rancher_list_local_repos prints out all of the local
# repos to STDOUT.
rancher_list_local_repos() {
    __RANCHER_PATH="${GOPATH}/src/github.com/rancher"
    __IMAGE_BUILD_REPOS=$(find "${__RANCHER_PATH}" \
        -path "*image-build-*" \
        -type f -not -path "*image-build-tools/*" \
        -type f -not -path "*image-build-base/*" \
        -type f -name "Dockerfile")

    for i in ${__IMAGE_BUILD_REPOS}; do
        echo "$i" | sed 's/\/Dockerfile//g'
    done
}

# rke2_list_docker_images prints out all the base and rke2 docker images
rke2_list_docker_images() {
    curl -s 'https://raw.githubusercontent.com/rancher/rke2/master/developer-docs/image_sources.md' | \
        grep -E 'rancher/(hardened-build-base|rke2)' | awk -F '|' '{print $4}' | xargs -n1 echo | \
        sort | uniq
}

# rke2_fecth_chart_index prints out rke2-chasrts index.yaml file
rke2_charts_get_index() {
    curl -sL https://raw.githubusercontent.com/rancher/rke2-charts/main/index.yaml
}
