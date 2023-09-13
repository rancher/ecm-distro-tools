#!/bin/sh

set -e

. libstd-ecm.sh

usage() {
    echo "usage: $0 [h] <tag> <stable_or_latest>
    -h    show help
    -d    dry run mode

examples:
    $0 v2.7.1 v2.8.3
    $0 -d"
}

while getopts 'hd' c; do
    case $c in
    h)
        usage
        exit 0
    ;;
    d)
        DRY_RUN=true
    ;;
    *)
        usage
        exit 1
    ;;
    esac
done

has_drone

if [ $# -ne 2 ]; then
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

    BUILD_NUMBER=$(drone build ls rancher/rancher --page ${PAGE} --event tag --format "{{.Number}},{{.Ref}}"| grep ${SOURCE_TAG}$ |cut -d',' -f1|head -1)
    if [ ! -n ${BUILD_NUMBER} ]; then
        echo "error: no build found for tag: ${SOURCE_TAG}"
        exit 1
    fi
    
    PAGE=$((PAGE+1))

    sleep 1
done

echo "Found build number ${BUILD_NUMBER} for tag ${SOURCE_TAG}"
if [ ${DRY_RUN} = true ]; then
    CMD="drone build promote rancher/rancher ${BUILD_NUMBER} promote-docker-image --param=SOURCE_TAG=${SOURCE_TAG} --param=DESTINATION_TAG=${DESTINATION_TAG}"
    echo "${CMD}"
elif
    #drone build promote rancher/rancher ${BUILD_NUMBER} promote-docker-image --param=SOURCE_TAG=${SOURCE_TAG} --param=DESTINATION_TAG=${DESTINATION_TAG}
    BUILD_NUMBER=""
fi

echo "promoting Chart${SOURCE_TAG} to ${DESTINATION_TAG}"

BUILD_NUMBER=$(drone build ls rancher/rancher --event tag --format "{{.Number}},{{.Ref}}"| grep ${SOURCE_TAG}$ |cut -d',' -f1|head -1)
if [ ! -n ${BUILD_NUMBER} ];then
    echo "error: no build found for tag: ${SOURCE_TAG}"
    exit 1
fi

if [ ${DRY_RUN} = true ]; then
    CMD="drone build promote rancher/rancher ${BUILD_NUMBER} promote-stable"
    echo "${CMD}"
elif
    #drone build promote rancher/rancher ${BUILD_NUMBER} promote-stable
fi

exit 0
