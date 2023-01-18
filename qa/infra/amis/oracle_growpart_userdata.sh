#!/bin/sh

# Update Partition
yum install cloud-utils-growpart -y

VOL_NAME=$(pvs | grep -i xvda)

if [ "${VOL_NAME}" ]; then
    growpart /dev/xvda 2
    pvresize /dev/xvda2
else
    growpart /dev/nvme0n1 2
    pvresize /dev/nvme0n1p2
fi

lvextend -r -l +100%FREE /dev/mapper/ol-root
xfs_growfs /
