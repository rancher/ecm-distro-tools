#!/bin/sh

get_cili() {
    CILIUM_CLI_VERSION=$(curl -s https://raw.githubusercontent.com/cilium/cilium-cli/master/stable.txt)
CLI_ARCH=amd64
if [ "$(uname -m)" = "aarch64" ]; then CLI_ARCH=arm64; fi
curl -L --fail --remote-name-all https://github.com/cilium/cilium-cli/releases/download/${CILIUM_CLI_VERSION}/cilium-linux-${CLI_ARCH}.tar.gz{,.sha256sum}
sha256sum --check cilium-linux-${CLI_ARCH}.tar.gz.sha256sum
sudo tar xzvfC cilium-linux-${CLI_ARCH}.tar.gz /usr/local/bin
rm cilium-linux-${CLI_ARCH}.tar.gz{,.sha256sum}

}

# --- install helm to the node ---
get_helm() {
    has_bin curl
    sudo curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3
    sudo chmod +x get_helm.sh
    sudo ./get_helm.sh
}

# --- install docker ---
get_docker() {
    has_bin curl
    curl -fsSL https://get.docker.com -o get-docker.sh && sudo chmod +x get-docker.sh
    sudo ./get-docker.sh
    has_bin docker
    sudo systemctl enable docker --now
}

# --- install wireguard ---
get_wireguard() {
    printf "use your package manager to update && upgrade, then install wireguard \n note there are more steps to get wireguard working on SLES and SLEM"
}

# --- install etcdctl ---
get_etcdctl() {
    has_bin curl
    _etcd_version=v3.5.0
    # choose either URL
    # GOOGLE_URL=https://storage.googleapis.com/etcd
    _github_url=https://github.com/etcd-io/etcd/releases/download
    _download_url=${_github_url}

    rm -f /tmp/etcd-${_etcd_version}-linux-amd64.tar.gz
    rm -rf /tmp/etcd-download-test && mkdir -p /tmp/etcd-download-test
    curl -L ${_download_url}/${_etcd_version}/etcd-${_etcd_version}-linux-amd64.tar.gz -o /tmp/etcd-${_etcd_version}-linux-amd64.tar.gz
    tar xzvf /tmp/etcd-${_etcd_version}-linux-amd64.tar.gz -C /tmp/etcd-download-test --strip-components=1
    rm -f /tmp/etcd-${_etcd_version}-linux-amd64.tar.gz
    /tmp/etcd-download-test/etcd --version
    /tmp/etcd-download-test/etcdctl version
    /tmp/etcd-download-test/etcdutl version
    sudo cp /tmp/etcd-download-test/etcdctl /usr/bin/etcdctl
    etcdctl version
}

# --- install zerotier vpn ---
get_zt() {
    has_bin curl
    curl -s https://install.zerotier.com | sudo bash
    wait
    sudo zerotier-cli join YOUR_ZT_NETWORK_ID
}

# --- install nats.io ---
get_nats() {
    has_bin wget
    wget https://github.com/nats-io/nats-server/releases/download/v2.9.11/nats-server-v2.9.11-linux-amd64.tar.gz
    sudo tar -zxf nats-server-v2.9.11-linux-amd64.tar.gz
    sudo cp nats-server-v2.9.11-linux-amd64/nats-server /usr/bin/
    nats-server -v
}

# --- install krew plugin for kubectl ---
get_krew() {
    has_bin git
    has_bin kubectl
    (
  set -x; cd "$(mktemp -d)" &&
  OS="$(uname | tr '[:upper:]' '[:lower:]')" &&
  ARCH="$(uname -m | sed -e 's/x86_64/amd64/' -e 's/\(arm\)\(64\)\?.*/\1\2/' -e 's/aarch64$/arm64/')" &&
  KREW="krew-${OS}_${ARCH}" &&
  curl -fsSLO "https://github.com/kubernetes-sigs/krew/releases/latest/download/${KREW}.tar.gz" &&
  tar zxvf "${KREW}.tar.gz" &&
  ./"${KREW}" install krew
)
}

# --- install krew plugin for kubectl ---
get_kuttl() {
    has_bin kubectl
    kubectl krew install kuttl
}

# --- install akri into the cluster using helm ---
get_akri() {
    has_bin helm
    helm repo add akri-helm-charts https://project-akri.github.io/akri/
    helm install akri akri-helm-charts/akri --set kubernetesDistro=k3s
}

# --- install wasmcloud into the cluster using helm ---
get_wasmcloud() {
    has_bin helm
    helm repo add wasmcloud https://wasmcloud.github.io/wasmcloud-otp/
    helm install wasmcloud wasmcloud/wasmcloud-host

}

get_sono() {
    has_bin wget
    _arch
    arch=$(if [ "$(uname -m)" = "x86_64" ]; then echo "amd64"; else echo "arm64"; fi)
    wget https://github.com/vmware-tanzu/sonobuoy/releases/download/v0.56.16/sonobuoy_0.56.16_linux_"${arch}".tar.gz
    sudo tar -xzf sonobuoy_0.56.16_linux_amd64.tar.gz -C /usr/local/bin
}
