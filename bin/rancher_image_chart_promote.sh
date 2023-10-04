#!/bin/sh

set -e

. libstd-ecm.sh

usage() {
    echo "usage: $0 [h] <tag> <stable_or_latest>
    -h    show help

examples:
    $0 v2.7.1 latest"
}

while getopts 'h' c; do
    case $c in
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

has_drone

if [ $# -lt 2 ]; then
    echo "error: $0 <tag> <stable_or_latest>"
    exit 1
fi

SOURCE_TAG=$1

case $2 in 
    "stable")
    ;;
    "latest")
    ;;
    "donotuse")
    ;;
    "*")
        echo "error: tag needs to be stable, latest, or donotuse for testing. given: $2"
    ;;
esac
DESTINATION_TAG=$2

BUILD_NUMBER=""
PAGE=1

until [ ${PAGE} -gt 100 ]; do
    echo "finding build number for tag: ${SOURCE_TAG}"

    BUILD_NUMBER=$(drone build ls rancher/rancher --page ${PAGE} --event tag --format "{{.Number}},{{.Ref}}"| grep "${SOURCE_TAG}"$ |cut -d',' -f1|head -1)
    if [ -n "${BUILD_NUMBER}" ]; then
        break
    fi
    
    PAGE=$((PAGE+1))

    sleep 1
done

if [ ! -n "${BUILD_NUMBER}" ]; then
    echo "error: no build found for tag: ${SOURCE_TAG}"
    exit 1
fi

echo "Found build number ${BUILD_NUMBER} for tag ${SOURCE_TAG}"

drone build promote rancher/rancher "${BUILD_NUMBER} promote-docker-image --param=SOURCE_TAG=${SOURCE_TAG} --param=DESTINATION_TAG=${DESTINATION_TAG}"
BUILD_NUMBER=""
sleep 2

echo "promoting Chart${SOURCE_TAG} to ${DESTINATION_TAG}"

BUILD_NUMBER=$(drone build ls rancher/rancher --event tag --format "{{.Number}},{{.Ref}}"| grep "${SOURCE_TAG}"$ |cut -d',' -f1|head -1)
if [ ! -n "${BUILD_NUMBER}" ]; then
    echo "error: no build found for tag: ${SOURCE_TAG}"
    exit 1
fi

drone build promote rancher/rancher "${BUILD_NUMBER}" promote-stable

exit 0
