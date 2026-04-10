#!/bin/sh

# Use rocky's repo mirrors for Suse Liberty 8.9 setup
# Refer: https://github.com/k3s-io/k3s/issues/10367#issuecomment-2181517372
RELEASE_VER="$1"
BASE_ARCH="$2"
CONTENT_DIR="pub/rocky"
APPSTREAM_FILE="/tmp/repo_content_file"
DEVEL_FILE="/tmp/repo_content_file"

echo "[appstream]
name=Rocky Linux ${RELEASE_VER} - AppStream
mirrorlist=https://mirrors.rockylinux.org/mirrorlist?arch=${BASE_ARCH}&repo=AppStream-${RELEASE_VER}
#baseurl=http://dl.rockylinux.org/${CONTENT_DIR}/${RELEASE_VER}/AppStream/${BASE_ARCH}/os/
gpgcheck=0
enabled=1" > "${APPSTREAM_FILE}"

cat "${APPSTREAM_FILE}" | sudo tee -a /etc/yum.repos.d/Appstream.repo

echo "[develrepo]
name=Rocky Linux ${RELEASE_VER} - Devel
mirrorlist=https://mirrors.rockylinux.org/mirrorlist?arch=${BASE_ARCH}&repo=rocky-devel-${RELEASE_VER}
#baseurl=https://dl.rockylinux.org/${CONTENT_DIR}/${RELEASE_VER}/devel/${BASE_ARCH}/os/
gpgcheck=0
enabled=1" > "${DEVEL_FILE}"

cat "${DEVEL_FILE}" | sudo tee -a /etc/yum.repos.d/Devel.repo
