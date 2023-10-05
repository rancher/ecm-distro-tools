#!/bin/sh

set -e

TMP_DIR=""
REPO_NAME="ecm-distro-tools"
REPO_URL="https://github.com/rancher/${REPO_NAME}"
REPO_RELEASE_URL="${REPO_URL}/releases"
INSTALL_DIR="/usr/local/bin/ecm-distro-tools"
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
    TMP_HASH=${TMP_DIR}/ecm-distro-tools.hash
    TMP_BIN=${TMP_DIR}/ecm-distro-tools.bin
    cleanup() {
        code=$?
        set +e
        trap - EXIT
        rm -rf ${TMP_DIR}
        exit $code
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
        curl -o "$1" -fsSL "$2"
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
    TARBALL_URL="${REPO_RELEASE_URL}/download/${RELASE_VERSION}/ecm-distro-tools.${SUFFIX}.tar.gz"

    echo "downloading tarball from ${TARBALL_URL}"
    
    download "${TMP_DIR}" "$1"
}

# install_binaries installs the binaries from the downloaded tar.
install_binaries() {
    cd "${TMP_DIR}"
    tar zxvf "${TMP_DIR}/$1"
    
    find . -type f -name "*.${SUFFIX} -exec cp {} ${INSTALL_DIR}" \;
}

{ # main
    if [ -z "$1" ]; then 
        echo "error: release version required"
        exit 1
    fi
    RELASE_VERSION=$1

    echo "Installing ECM Distro Tools: ${RELASE_VERSION}"

    setup_tmp
    setup_arch

    verify_downloader curl || verify_downloader wget || fatal "error: cannot find curl or wget"
    download_tarball "${RELEASE_TARBALL}"
    install_binaries "${RELEASE_TARBALL}"

    printf "Run command to access tools:\n\nPATH=%s:%s" "${PATH}" "${INSTALL_DIR}"

    exit 0
}



