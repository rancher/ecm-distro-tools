package repository

import (
	"errors"
	"strings"
)

// RKE2HardenedImages
var RKE2HardenedImages = []string{
	"rancher/image-build-base",
	"rancher/image-build-calico",
	"rancher/image-build-cni-plugins",
	"rancher/image-build-containerd",
	"rancher/image-build-coredns",
	"rancher/image-build-crictl",
	"rancher/image-build-dns-nodecache",
	"rancher/image-build-etcd",
	"rancher/image-build-flannel",
	"rancher/image-build-ib-sriov-cni",
	"rancher/image-build-k8s-metrics-server",
	"rancher/image-build-kubernetes",
	"rancher/image-build-multus",
	"rancher/image-build-rke2-cloud-provider",
	"rancher/image-build-runc",
	"rancher/image-build-sriov-cni",
	"rancher/image-build-sriov-network-device-plugin",
	"rancher/image-build-sriov-network-resources-injector",
	"rancher/image-build-sriov-operator",
	"rancher/image-build-whereabouts",
	"rancher/ingress-nginx",
}

// RKE2MirroredImages
var RKE2MirroredImages = []string{
	"mirrored-ingress-nginx-kube-webhook-certgen",
	"mirrored-cilium-cilium",
	"mirrored-cilium-operator-aws",
	"mirrored-cilium-operator-azure",
	"mirrored-cilium-operator-generic",
	"mirrored-calico-operator",
	"mirrored-calico-ctl",
	"mirrored-calico-kube-controllers",
	"mirrored-calico-typha",
	"mirrored-calico-node",
	"mirrored-calico-pod2daemon-flexvol",
	"mirrored-calico-cni",
	"mirrored-calico-apiserver",
	"mirrored-cloud-provider-vsphere-cpi-release-manager",
	"mirrored-cloud-provider-vsphere-csi-release-driver",
	"mirrored-cloud-provider-vsphere-csi-release-syncer",
	"mirrored-sig-storage-csi-node-driver-registrar",
	"mirrored-sig-storage-csi-resizer",
	"mirrored-sig-storage-livenessprobe",
	"mirrored-sig-storage-csi-attacher",
	"mirrored-sig-storage-csi-provisioner",
}

// RKE2Adjacent
var RKE2Adjacent = []string{
	"rancher/rke2-upgrade",
	"rancher/rke2-packaging",
	"rancher/system-agent-installer-rke2",
	"rancher/system-upgrade-controller",
}

const ownerRepoSeparattor = "/"

func SplitOwnerRepo(ownerRepo string) (string, string, error) {
	if !strings.Contains(ownerRepo, ownerRepoSeparattor) {
		return "", "", errors.New("invalid format")
	}

	ss := strings.Split(ownerRepo, ownerRepoSeparattor)
	if len(ss) != 2 {
		return "", "", errors.New("invalid format")
	}

	return ss[0], ss[1], nil
}
