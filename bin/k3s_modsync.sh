#!/bin/sh

K8S_REPO="kubernetes/kubernetes"
K8S_COMMIT="${K8S_COMMIT:-}"
if [ -z "${K8S_COMMIT}" ]; then
    K8S_REPLACE=$(go mod edit --json | jq -r '.Replace[] | select(.Old.Path | contains("k8s.io/kubernetes")) | .New.Path + " " + .New.Version')
    if [ -n "${K8S_REPLACE}" ]; then
        K8S_REPO=$(echo "${K8S_REPLACE#github.com/}" | awk '{print $1}')
        K8S_VERSION=$(echo "${K8S_REPLACE#github.com/}" | awk '{print $2}')
    else
        K8S_VERSION=$(go mod edit --json | jq -r '.Require[] | select(.Path | contains("k8s.io/kubernetes")) | .Version')
    fi
    echo "Updating go.mod replacements from ${K8S_REPO} ${K8S_VERSION}"
    K8S_COMMIT=$(echo "${K8S_VERSION}" | grep -oE '\w{12}$')
    if [ -z "${K8S_COMMIT}" ]; then
        K8S_COMMIT=${K8S_VERSION}
    fi
else
    echo "Updating go.mod replacements from ${K8S_REPO} ${K8S_COMMIT}"
fi

K8S_GO_MOD=$(curl -qsL "https://raw.githubusercontent.com/${K8S_REPO}/${K8S_COMMIT}/go.mod")

# update replacements
go mod edit --json | jq -r '.Replace[] | .Old.Path + " " + .New.Path + " " + .New.Version' |
while read -r OLDPATH NEWPATH VERSION; do
    REPLACEMENT=$(echo "${K8S_GO_MOD}" | go mod edit --json /dev/stdin | jq -r --arg OLDPATH "${OLDPATH}" '.Replace[] | select(.Old.Path==$OLDPATH) | .New.Version')
    echo "Checking for updates to ${OLDPATH} ${VERSION} -> ${REPLACEMENT}"
    if [ -n "${REPLACEMENT}" ] && [ "${REPLACEMENT}" != "null" ] && echo "${NEWPATH}" | grep -vq k3s && semver-cli greater "${REPLACEMENT}" "${VERSION}" ; then
        (set -x; go mod edit --replace="${OLDPATH}=${NEWPATH}@${REPLACEMENT}")
    fi
done
