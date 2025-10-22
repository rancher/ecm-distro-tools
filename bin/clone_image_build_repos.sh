#!/bin/sh

set -e

. libstd-ecm.sh

usage() {
    echo "usage: $0 [dh]
    -d    dry run
    -h    show help

examples:
    $0 -d"
}

while getopts 'dh' c; do
    case $c in
    d)
        DRY_RUN=true
    ;;
    h)
        usage
        exit 0
    ;;
    *)
        usage
        exit 1
    ;;
    esac
done

REPOS="rancher/image-build-base
    rancher/image-build-calico
    rancher/image-build-cluster-proportional-autoscaler
    rancher/image-build-cni-plugins
    rancher/image-build-containerd
    rancher/image-build-coredns
    rancher/image-build-crictl
    rancher/image-build-dns-nodecache
    rancher/image-build-etcd
    rancher/image-build-flannel
    rancher/image-build-ib-sriov-cni
    rancher/image-build-k8s-metrics-server
    rancher/image-build-kubernetes
    rancher/image-build-multus
    rancher/image-build-rke2-cloud-provider
    rancher/image-build-runc
    rancher/image-build-sriov-cni
    rancher/image-build-sriov-network-device-plugin
    rancher/image-build-sriov-network-resources-injector
    rancher/image-build-sriov-operator
    rancher/image-build-whereabouts
    rancher/ingress-nginx"

has_git

for i in ${REPOS}; do
    repo=$(echo ${i} | awk -F '/' '{print $2}')

    if [ "${DRY_RUN}" ]; then
        echo "git clone git@github.com:${i}.git ../../${repo}"
    else
        git clone --depth 1 git@github.com:${i}.git ../../"${repo}"
    fi
done

exit 0
