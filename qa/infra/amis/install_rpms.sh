#!/bin/sh

yum install -y container-selinux
rpm -i https://github.com/k3s-io/k3s-selinux/releases/download/v1.2.stable.2/k3s-selinux-1.2-2.el8.noarch.rpm
