#!/bin/sh

set -e

TMP_DIR=""
REPO_NAME="ecm-distro-tools"
REPO_URL="https://github.com/rancher/${REPO_NAME}"
REPO_RELEASE_URL="${REPO_URL}/releases"
INSTALL_DIR="$HOME/.local/bin/ecm-distro-tools"
SUFFIX=""
DOWNLOADER=""


# setup_arch set arch and suffix fatal if architecture not supported.
setup_arch() {
    case $(uname -m) in
    x86_64|amd64)
        ARCH=amd64
        SUFFIX=$(uname -s | tr '[:upper:]' '[:lower:]')-${ARCH}
        ;;
    aarch64|arm64)
        ARCH=arm64
        SUFFIX=$(uname -s | tr '[:upper:]' '[:lower:]')-${ARCH}
        ;;
    *)
        fatal "unsupported architecture ${ARCH}"
        ;;
    esac
}

# setup_tmp creates a temporary directory and cleans up when done.
setup_tmp() {
    TMP_DIR=$(mktemp -d -t ecm-distro-tools-install.XXXXXXXXXX)
    cleanup() {
        code=$?
        set +e
        trap - EXIT
        rm -rf "${TMP_DIR}"
        exit "$code"
    }
    trap cleanup INT EXIT
}

# verify_downloader verifies existence of network downloader executable.
verify_downloader() {
    cmd="$(command -v "${1}")"
    if [ -z "${cmd}" ]; then
        return 1
    fi
    if [ ! -x "${cmd}" ]; then
        return 1
    fi

    DOWNLOADER=${cmd}
    return 0
}

# download downloads a file from a url using either curl or wget.
download() {
    case "${DOWNLOADER}" in
    *curl)
        cd "$1" && { curl -fsSLO "$2" ; cd -; }
    ;;
    *wget)
        wget -qO "$1" "$2"
    ;;
    esac

    if [ $? -ne 0 ]; then
        echo "error: download failed"
        exit 1
    fi
}

# download_tarball downloads the tarbal for the given version.
download_tarball() {
    TARBALL_URL="${REPO_RELEASE_URL}/download/${RELEASE_VERSION}/ecm-distro-tools.${SUFFIX}.tar.gz"

    echo "downloading tarball from ${TARBALL_URL}"
    
    download "${TMP_DIR}" "${TARBALL_URL}"
}

# install_binaries installs the binaries from the downloaded tar.
install_binaries() {
    cd "${TMP_DIR}"
    tar -xf "${TMP_DIR}/ecm-distro-tools.${SUFFIX}.tar.gz"
    rm "${TMP_DIR}/ecm-distro-tools.${SUFFIX}.tar.gz"
    mkdir -p "${INSTALL_DIR}"

    for f in * ; do
      file_name="${f}"
      if echo "${f}" | grep -q "${SUFFIX}"; then
        file_name=${file_name%"-${SUFFIX}"}
      fi
      cp "${TMP_DIR}/${f}" "${INSTALL_DIR}/${file_name}"
    done
}

{ # main
    RELEASE_VERSION=$1
    if [ -n "${ECM_VERSION}" ]; then
        RELEASE_VERSION=${ECM_VERSION}
    fi

    if [ -z "$RELEASE_VERSION" ]; then 
        RELEASE_VERSION=$(basename $(curl -Ls -o /dev/null -w %\{url_effective\} https://github.com/rancher/ecm-distro-tools/releases/latest))
    fi

    echo "Installing ECM Distro Tools: ${RELEASE_VERSION}"

    setup_tmp
    setup_arch

    verify_downloader curl || verify_downloader wget || fatal "error: cannot find curl or wget"
    download_tarball
    install_binaries

    printf "Run command to access tools:\n\nPATH=%s:%s\n\n" "${PATH}" "${INSTALL_DIR}"

    exit 0
}
