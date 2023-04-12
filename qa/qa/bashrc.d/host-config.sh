#!/bin/sh

get_report() {
    _product="${1:-$PRODUCT}"
    cat << EOF >> validation_template.md
<!-- Thanks for using this template. Comment like this will be hidden. Enjoy! -->
<!-- Make sure you remove any sensitive information and change IPs before sharing. -->
##Environment Details
VERSION=${VERSION}
COMMIT=${COMMIT}

*Infrastructure*
- [X] Cloud
- [ ] Hosted 

*Node(s) CPU architecture, OS, and version:*

$(uname -rpos) 
$(grep /etc/os-release -i -e pretty)
 
*Cluster Configuration:*

$(kubectl get nodes) 

*Config.yaml:*

$(sudo cat /etc/rancher/"${PRODUCT}"/config.yaml)

<details><summmary><h4> YOUR_REPRODUCED_RESULTS_HERE </h4></summary>

<!-- Provide the command to install "${_product}" -->

 curl https://get.${_product}.io --output install-"${_product}".sh
 sudo chmod +x install-"${_product}".sh
 sudo groupadd --system etcd && sudo useradd -s /sbin/nologin --system -g etcd etcd
 sudo modprobe ip_vs_rr
 sudo modprobe ip_vs_wrr
 sudo modprobe ip_vs_sh
 sudo printf "on_oovm.panic_on_oom=0 \nvm.overcommit_memory=1 \nkernel.panic=10 \nkernel.panic_ps=1 \nkernel.panic_on_oops=1 \n" > ~/60-rke2-cis.conf or ~/90-kubelet.conf
 sudo cp 60-rke2-cis.conf /etc/sysctl.d/ or 90-kubelet.conf
 sudo systemctl restart systemd-sysctl



"$(history | cut -c 8-)"

**Results:**

</details>

<details><summmary><h4> YOUR_VALIDATION_RESULTS_HERE </h4></summary>
 
## Validation Steps
 
- Install "${_product}":
 
 
**Results:** 

**Additional context / logs:**
</details>  
EOF
}

set_nats() {
cat <<'EOF' >> nats.config
port: 4222
net: '0.0.0.0'
authorization: {
  users: [
    {user: "k3s", password: "EXAMPLE_PASS"},
    {user: "rke2", password: "EXAMPLE_PASS"}
  ]
}
EOF
sudo mkdir -p /srv/nats/
sudo cp nats.config /srv/nats/nats.config
cat <<'EOF' >> nats.service
[Unit]
Description=NATS jetstream messaging server

[Service]
ExecStart="/usr/bin/nats-server -js -c /srv/nats/nats.config"
User=nats
Restart=on-failure

[Install]
WantedBy=multi-user.target
EOF
sudo cp nats.service /etc/systemd/system/nats.service
sudo adduser --system --group --no-create-home --shell /bin/false nats
sudo systemctl enable nats --now
}



get_rootless() {
    if [ "$USER" != "ubuntu" ]; then
        printf "I haven't mapped this process out on different OS's yet."
        get_help get_rootless
    elif [ ! "$(command -v newuidmap)" ]; then
        printf "You'll need to install uidmap ie: \n sudo apt install uidmap\n"
    else
        wget https://raw.githubusercontent.com/k3s-io/k3s/master/k3s-rootless.service
        mkdir -p /home/ubuntu/.config/systemd/user/
        cp k3s-rootless.service /home/ubuntu/.config/systemd/user/k3s-rootless.service
        printf "[Service]\nDelegate=cpu cpuset io memory pids\n" > delegate.conf
        printf "net.ipv4.ip_forward=1\nnet.ipv6.conf.all.forwarding=1\n" | sudo tee -a /etc/sysctl.conf /dev/null
        sudo mkdir -p /etc/systemd/system/user@.service.d/
        sudo cp ~/delegate.conf /etc/systemd/system/user@.service.d/delegate.conf
        sudo tee -a /etc/modules <<EOF
fuse
tun
tap 
bridge
br_netfilter 
veth
ip_tables
ip6_tables
iptable_nat
ip6table_nat
iptable_filter
ip6table_filter
nf_tables
x_tables
xt_MASQUERADE
xt_addrtype
xt_comment
xt_conntrack
xt_mark
xt_multiport
xt_nat
xt_tcpudp
EOF
        #printf "systemd.unified_cgroup_hierarchy=1" | sudo tee -a /etc/default/grub > /dev/null 
        #sed no worky
        #sed -i 's/GRUB_CMDLINE_LINUX_DEFAULT="[^"]*/& systemd.unified_cgroup_hierarchy=1/' /etc/default/grub
        #sudo sysctl --system
        #systemctl --user daemon-reload
        #sudo update-grub
        printf "you'll need to update /etc/default/grub and add systemd.unified_cgroup_hierarchy=1 to GRUB_CMDLINE_LINUX_DEFAULT=""\n"
        printf "You'll need to update your packages then install uidmap and reboot to finish the setup.\n"
    fi
}

set_harden() {
    _product="${1:-$PRODUCT}"
    case "${_product}" in
    rke2) printf "on_oovm.panic_on_oom=0 \nvm.overcommit_memory=1 \nkernel.panic=10 \nkernel.panic_ps=1 \nkernel.panic_on_oops=1 \nkernel.keys.root_maxbytes=25000000" > ~/60-rke2-cis.conf
          sudo cp 60-rke2-cis.conf /etc/sysctl.d/
          sudo sysctl -p /etc/sysctl.d/60-rke2-cis.conf
            ;;
    k3s) printf "on_oovm.panic_on_oom=0 \nvm.overcommit_memory=1 \nkernel.panic=10 \nkernel.panic_ps=1 \nkernel.panic_on_oops=1 \nkernel.keys.root_maxbytes=25000000" > ~/90-kubelet.conf
         sudo cp 90-kubelet.conf /etc/sysctl.d/
         sudo sysctl -p /etc/sysctl.d/90-kubelet.conf
            ;;
    esac
    sudo modprobe ip_vs_rr
    sudo modprobe ip_vs_wrr
    sudo modprobe ip_vs_sh
    wait
    sudo systemctl restart systemd-sysctl
}

set_etcduser() {
    sudo groupadd --system etcd && sudo useradd -s /sbin/nologin --system -g etcd etcd
}

set_figs() {
    _product="${1:-$PRODUCT}"
    sudo mkdir -p /etc/rancher/"${_product}"/;
    #sudo cat <<EOF >> "${_product}"-config.yaml
    cat <<EOF >> "${_product}"-config.yaml
server: https://
write-kubeconfig-mode: 644
debug: true
token: YOUR_TOKEN_HERE
cni: multus,cilium
profile: cis-1.6
selinux: true

#protect-kernel-defaults: true
#cluster-init: true

#disable: rke2-ingress-nginx
#pod-security-admission-config-file: "/etc/rancher/rke2/base-pss.yaml"

#datastore-endpoint: etcd, Mysql, Postgres, Sqlite, nats-jetstream
##EDIT these
node-ip: $(hostname -I)
#node-external-ip: IPV4,IPV6
##DUALSTACK note reverse the IPV4/IPV6 order to prioritize IPV6 over IPV4 if needed during testing
#cluster-cidr: 10.42.0.0/16,2001:cafe:42:0::/56
#service-cidr: 10.43.0.0/16,2001:cafe:42:1::/112
#multi-cluster-cidr: true

##KUBELET ARGS 
#kubelet-arg:
#  - alsologtostderr=true
#  - feature-gates=MemoryManager=true
#  - kube-reserved=cpu=400m,memory=1Gi
#  - system-reserved=cpu=400m,memory=1Gi
#  - memory-manager-policy=Static
#  - reserved-memory=0:memory=2Gi
#  - port=11250

##KUBELET APISERVER ARGS
#kube-apiserver-arg:
#  - tls-cipher-suites=TLS_CHACHA20_POLY1305_SHA256,TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256

##SNAPSHOTS
#snapshot-compress: true
#etcd-snapshot-retention: 18
#etcd-snapshot-schedule-cron: "*/3 * * * *" #every 3 minutes

##VSPHERE
#private-registry: "/etc/rancher/rke2/registries.yaml"
#cloud-provider-name: "rancher-vsphere"
#cloud-provider-config: /home/rancher/vsphere.conf

#enable-pprof: true
#secrets-encryption: true
#kube-proxy-arg: "proxy-mode=ipvs"

##ETCD S3 CONFIG
#etcd-s3: true
#etcd-s3-bucket: "YOUR_BUCKET_NAME"
#etcd-s3-folder: "snapshotrestore"
#etcd-s3-region: "YOUR_BUCKET_REGION"
#etcd-s3-endpoint: "YOUR_S3_ENDPOINT"
#etcd-s3-access-key: "YOUR_ACCESS_KEY"
#etcd-s3-secret-key: "YOUR_SECRET_KEY"

#flannel-ipv6-masq: true
#disable: rke2-ingress-nginx

## SPLIT ROLES

##--etcd-only---
#disable-apiserver: true
#disable-controller-manager: true
#disable-scheduler: true
#node-taint:
#  - node-role.kubernetes.io/etcd:NoExecute

##--etcd-cp---
#node-taint:
#  - node-role.kubernetes.io/control-plane:NoSchedule
#  - node-role.kubernetes.io/etcd:NoExecute

##--etcd-worker---
#disable-apiserver: true
#disable-controller-manager: true
#disable-scheduler: true

##--cp-only---
#disable-etcd: true
#node-taint:
#  - node-role.kubernetes.io/control-plane:NoSchedule

##--cp-worker---
#disable-etcd: true

EOF
    sudo cp "${_product}"-config.yaml /etc/rancher/"${_product}"/config.yaml;
}

# --- set registries to look for bci hardened images ---
set_registries() {
    cat <<'EOF' >> registries.yaml
    mirrors:
    docker.io:
        rewrite:
        "^rancher/hardened-etcd(.*)": "bcibase/hardened-etcd$1"
        "^rancher/hardened-kubernetes(.*)": "bcibase/hardened-kubernetes$1"
        "^rancher/hardened-rke2-runtime(.*)": "bcibase/hardened-rke2-runtime$1"
        "^rancher/nginx-ingress-controller(.*)": "bcibase/nginx-ingress-controller$1"
        "^rancher/nginx-ingress-controller-chroot(.*)": "bcibase/nginx-ingress-controller-chroot$1"
        "^rancher/hardened-calico(.*)": "bcibase/hardened-calico$1"
        "^rancher/hardened-sriov-network-resources-injector(.*)": "bcibase/hardened-sriov-network-resources-injector$1"
        "^rancher/hardened-k8s-metrics-server(.*)": "bcibase/hardened-k8s-metrics-server$1"
        "^rancher/hardened-sriov-network-webhook(.*)": "bcibase/hardened-sriov-network-webhook$1"
        "^rancher/hardened-sriov-network-operator(.*)": "bcibase/hardened-sriov-network-operator$1"
        "^rancher/hardened-sriov-network-device-plugin(.*)": "bcibase/hardened-sriov-network-device-plugin$1"
        "^rancher/hardened-flannel(.*)": "bcibase/hardened-flannel$1"
        "^rancher/hardened-crictl(.*)": "bcibase/hardened-crictl$1"
        "^rancher/hardened-ib-sriov-cni(.*)": "bcibase/hardened-ib-sriov-cni$1"
        "^rancher/hardened-runc(.*)": "bcibase/hardened-runc$1"
        "^rancher/hardened-rke2-cloud-provider(.*)": "bcibase/hardened-rke2-cloud-provider$1"
        "^rancher/hardened-cni-plugins(.*)": "bcibase/hardened-cni-plugins$1"
        "^rancher/hardened-dns-node-cache(.*)": "bcibase/hardened-dns-node-cache$1"
        "^rancher/hardened-containerd(.*)": "bcibase/hardened-containerd$1"
        "^rancher/hardened-cluster-autoscaler(.*)": "bcibase/hardened-cluster-autoscaler$1"
        "^rancher/hardened-coredns(.*)": "bcibase/hardened-coredns$1"
        "^rancher/hardened-multus-cni(.*)": "bcibase/hardened-multus-cni$1"
        "^rancher/hardened-whereabouts(.*)": "bcibase/hardened-whereabouts$1"
EOF
    sudo cp registries.yaml /etc/rancher/rke2/registries.yaml
}

# --- vsphere in tree ---
set_vsphere() {
    cat <<'EOF' >> vsphere.conf
# vsphere.conf
Name:         vsphere-cloud-config
Namespace:    kube-system
Labels:       app.kubernetes.io/managed-by=Helm
              component=rancher-vsphere-cpi-cloud-controller-manager
              vsphere-cpi-infra=config
Annotations:  meta.helm.sh/release-name: rancher-vsphere-cpi
              meta.helm.sh/release-namespace: kube-system

Data
====
vsphere.yaml:
----
# Global properties in this section will be used for all specified vCenters unless overriden in VirtualCenter section.

[Global]
datacenters = "YOUR_DATA_CENTER"
insecure-flag = "1"
user = "YOUR_USER_NAME"
password = "YOUR_PASSWORD"
server = "YOUR_SERVER_NAME"
port = "YOUR_PORT"
secret-namespace: "kube-system" 
secret-name: "vsphere-cpi-creds"
 
[VirtualCenter "YOUR_SERVER_NAME"]
user = "YOUR_USER_NAME"
password = "YOUR_PASSWORD"
port = "YOUR_PORT"
datacenters = "YOUR_DATA_CENTER" 

[Workspace]
server = "YOUR_SERVER_NAME"
datacenters = "YOUR_DATA_CENTER"
default-datastore = "YOUR_DATA_STORE_URL"
resourcepool-path = "YOUR_RESOURCE_POOL_PATH"
folder = "YOUR_USERNAME"
EOF
}

# --- adding custom vsphere-values to the server manifests directory ---
set_vspheremanifests() {
    sudo mkdir -p /var/lib/rancher/rke2/server/manifests;
    cat <<'EOF' >> vsphere-values.yaml
# /var/lib/rancher/rke2/server/manifests/vsphere-values.yaml
apiversion: helm.cattle.io/v1
kind: HelmChartConfig
metadata:
  name: rancher-vsphere-cpi
  labels:
  namespace: kube-system
spec:
  valuesContent: |-
    vCenter:
      host: "YOUR_HOSTNAME"
      datacenters: "YOUR_DATA_CENTER"
      username: "YOUR_USERNAME"
      password: "YOUR_PASSWORD"
      credentialsSecret:
        generate: true
      labels:
        generate: true
        zone: "test-zone"
        region: "test-region"
    cloudControllerManager:
      nodeSelector:
        node-role.kubernetes.io/control-plane: "true"
---
apiversion: helm.cattle.io/v1
kind: HelmChartConfig
metadata:
  name: rancher-vsphere-csi
  namespace: kube-system
spec:
  valuesContent: |-
    vCenter:
      host: "YOUR_HOSTNAME"
      datacenters: "YOUR_DATA_CENTER"
      username: "YOUR_USERNAME"
      password: "YOUR_PASSWORD"
      clusterId: "YOUR_NODE_HOSTNAME"
      configSecret:
        configTemplate: |
         [Global]
         cluster-id = {{ required ".Values.vCenter.clusterId must be provided" (default .Values.vCenter.clusterId .Values.global.cattle.clusterId) | quote }}
         user = {{ .Values.vCenter.username | quote }}
         password = {{ .Values.vCenter.password | quote }}
         port = {{ .Values.vCenter.port | quote }}
         insecure-flag = {{ .Values.vCenter.insecureFlag | quote }}
         [VirtualCenter {{ .Values.vCenter.host | quote }}]
         datacenters = {{ .Values.vCenter.datacenters | quote }}
         [Labels]
         zone = "test-zone"
         region = "test-region"
    storageClass:
      datastoreURL: "YOUR_DATA_STORE_URL"
    csiController:
      nodeSelector:
        node-role.kubernetes.io/control-plane: "true"
EOF
    sudo cp vsphere-values.yaml /var/lib/rancher/rke2/server/manifests/vsphere-values.yaml
    cat <<'EOF' >> persistentVolume.yaml
apiversion: v1
kind: PersistentVolumeClaim
metadata:
  name: claim1
spec:
  accessModes:
    - ReadWriteOnce
  storageClassName: vsphere-csi-sc
  resources:
    requests:
      storage: 1Gi
EOF
    cat <<'EOF' >> useVolume-Workload.yaml
apiversion: "v1"
kind: "Pod"
metadata:
  name: "basic"
  labels:
    name: "basic"
spec:
  containers:
    - name: "basic"
      image: ranchertest/mytestcontainer:unprivileged
      ports:
        - containerPort: 8080
          name: "basic"
      volumeMounts:
        - mountPath: "/data"
          name: "pvol"
  volumes:
    - name: "pvol"
      persistentVolumeClaim:
        claimName: "claim1"
EOF
}

# --- creates a basic pod security standards file for k3s or rke2---
set_pss() {
    _product="${1:-$PRODUCT}"
    cat <<'EOF' >> base-pss.yaml
apiversion: apiserver.config.k8s.io/v1
kind: AdmissionConfiguration
plugins:
- name: PodSecurity
  configuration:
    apiversion: pod-security.admission.config.k8s.io/v1beta1
    kind: PodSecurityConfiguration
    defaults:
      enforce: "baseline"
      enforce-version: "latest"
    exemptions:
      usernames: []
      runtimeClasses: []
      namespaces: [kube-system, cis-operator-system, tigera-operator]
EOF
    sudo cp base-pss.yaml /etc/rancher/"${_product}"/base-pss.yaml
}

# --- sets up rhel 8-ish systems for rke2 ---
configure_rhel() {
  # Update the system packages
  sudo yum update -y

  # Install the dependencies for RKE2
  sudo yum install -y conntrack socat ebtables ethtool jq

  # Add the RKE2 RPM repository
  cat <<EOF | sudo tee /etc/yum.repos.d/rke2.repo
[rke2]
name=RKE2
baseurl=https://packages.rancher.com/rke2/rpm
enabled=1
gpgcheck=1
gpgkey=https://packages.rancher.com/gpg.key
EOF


  # Add the keyfile to fix NetworkManager
  sudo cp /etc/sysconfig/network-scripts/ifcfg-eth0 /etc/sysconfig/network-scripts/ifcfg-eth0.backup
  sudo sed -i '/^NM_CONTROLLED/d' /etc/sysconfig/network-scripts/ifcfg-eth0
  echo "NM_CONTROLLED=no" | sudo tee -a /etc/sysconfig/network-scripts/ifcfg-eth0 /dev/null

  # Disable nm-cloud-setup
  sudo systemctl disable nm-cloud-setup

}
