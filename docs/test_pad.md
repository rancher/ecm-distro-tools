# Test-Pad
The test-pad tool is designed to enable QA and developers to quickly deploy RKE2 and K3s clusters **locally**.  
This tool is built upon:
- vagrant
- libvirt (and the vagrant-libvirt plugin)
- RKE2/K3s E2E tests

**Note:** A similar tool, [Corral](https://github.com/rancherlabs/corral) enables consistent RKE2 and K3s deployments to Digital Ocean or AWS. If you want to deploy K3s or RKE2 on cloud resources, Corral may prove more useful. Currently only single node, and 3 node HA deployments are supported.

## Setup 
1) Download the latest version (currently 2.2.19) of Vagrant *from the website*. Do not use built-in packages, they often old or do not include the required ruby library extensions necessary to get certain plugins working.
2) Install libvirt/qemu on your host:  
    - [openSUSE](https://documentation.suse.com/sles/15-SP1/html/SLES-all/cha-vt-installation.html)
    - [ubuntu](https://ubuntu.com/server/docs/virtualization-libvirt)
    - [debian](https://wiki.debian.org/KVM#Installation)
    - [fedora](https://developer.fedoraproject.org/tools/virtualization/installing-libvirt-and-virt-install-on-fedora-linux.html)

The first time you use the tool, it will check and attempt to install the proper vagrant plugins.  
**Note:** The `vagrant-libvirt` plugin should be > v0.9.0. This solves several issues around networking and preventing VMs from being destroy if provisioning fails.

## Cluster Configuration
The followuing cluster configurations are supported:
- basic:        2 VMs, 1 server, 1 agent
- basic-lite:   1 VM,  1 server
- ha:           5 VMs, 3 servers, 2 agents
- ha-lite:      3 VMs, 3 servers
- split:        5 VMs, 3 etcd-only server, 2 cp-only servers. Taints on etcd and control-plane
- split-heavy:  7 VMs, 3 etcd-only server, 2 cp-only servers, 2 agents. Taints on etcd and control-plane
- split-lite:   3 VMs, 1 etcd-only server, 1 cp-only servers, 1 agent. Taints on etcd and control-plane
- rancher:      4 VMs, 1 single server with rancher, 3 blank VMs ready for provisioning

## Executable Version
There are 3 types of K3s or RKE2 that a user can deploy:
- A specific released version, such as `v1.22.9+k3s1` or `v1.24.1+rke2r2`.
- A COMMIT ID install, such as `763a8bc8fe376e3376faceed48c8c889d396b88c`.
- A local executable path, which enabled local dev/PR testing.

**Note:** For RKE2, regardless of Executable version, an airgap installation of RKE2 is conducted. Thus this 
tool can and should be used to test airgap specific issues. 

## VM Resources 
For K3s, each VM consumes 2 vcpus and 1GB of memory. For RKE2, each VM consumes 2 vcpus and 2GB of memory.  
Thus for a `split` cluster of RKE2, it is recommened to have a 8 core / 16 thread cpu and 16GB+ of memory.

By default, K3s VMs use Alpine, while RKE2 VMs use Ubuntu. Apline is a much smaller VM (500MB vs 1.5GB) and 
has a faster startup time. RKE2 must use Ubuntu because it only supports systemd. The default OS can be overriden
with the `-p [alpine|ubuntu]` flag.

## Examples:
- Deploy a 1 server, 1 agent cluster of K3s v1.22.9+k3s1.  
  `test-pad -r k3s -v v1.22.9+k3s1 -c basic`
-  Deploy a 1 server cluster of K3s v1.21.12+k3s1 with Rancher installed on top. Three blank VMs are also deployed.  
  `test-pad -r k3s -v v1.21.12+k3s1 -c rancher `
- Deploy a 1 etcd, 1 control-plane, 1 agent cluster of K3s commit build 1d4f995edd33186e178bfa9cf3d442dd244d2022  
  `test-pad -r k3s -v 1d4f995edd33186e178bfa9cf3d442dd244d2022 -c split-lite`  
- Deploy a 3 server cluster of a local build of K3s  
  `test-pad -r k3s -b ../../k3s/dist/artifacts/k3s -c ha-lite ` 
- Deploy a 3 server, 2 agent cluster of rke2 v1.23.5+rke2r1  
  `test-pad -r rke2 -v v1.23.5+rke2r1 -c ha`
- Deploy a 1 server and 1 agent cluser of a local build of rke2  
  `test-pad -r rke2 -b ../../rke2/dist/artifacts/rke2.linux-amd64.tar.gz -i ../../rke2/build/images -c basic`


## Typical Workflow
### Developers
You have a local code change/PR you want to test. The code change only affects HA configurations of RKE2.
- Build RKE2 binary and images:  
  `cd /PATH/TO/RKE2; make`
- Spin up the cluster:  
  `test-pad -r rke2 -b /PATH/TO/RKE2/dist/artifacts/rke2.linux-amd64.tar.gz -i /PATH/TO/RKE2/build/images -c ha`
- SSH into the individual nodes and realize that your code change has a problem:  
  `vagrant ssh server-0`
- Destroy the cluster:  
  `test-pad -d`
- Modify the RKE2 code and regenerate the binary  
  `cd /PATH/TO/RKE2; make package-bundle`
- Spin up the cluster again:  
  `test-pad -r rke2 -b /PATH/TO/RKE2/dist/artifacts/rke2.linux-amd64.tar.gz -i /PATH/TO/RKE2/build/images -c ha`


### QA
You have an K3s issue than is `To Test`, but validation requires a split role cluster.  
- Download the latest commit and test files:  
  `test-pad -r k3s -v <LATEST_COMMIT_FROM_MASTER> -c split --download`
- Modify the contents of `Vagrantfile` and add any additional configuration to the YAML section of the nodes as required.  
- Spin up the cluster:  
  `test-pad -r k3s -v <LATEST_COMMIT_FROM_MASTER> -c split --skip`
- SSH into the individual nodes and verify the issue as required:  
  `vagrant ssh server-cp-0`
- Destroy the cluster:  
  `test-pad -d`

If you need to modify the configuration again with different values:
- Kill the cluster:  
  `test-pad -k`
- Modify the contents of the `Vagrantfile` again.
- Spin up the cluster:  
  `test-pad -r k3s -v <LATEST_COMMIT_FROM_MASTER> -c split --skip`