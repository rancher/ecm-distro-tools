#!/bin/sh

cat <<EOF >> /etc/sysctl.d/60-rke2-cis.conf
vm.panic_on_oom=0
vm.overcommit_memory=1
kernel.panic=10
kernel.panic_on_oops=1
EOF
systemctl restart systemd-sysctl
useradd -r -c "etcd user" -s /sbin/nologin -M etcd -U
