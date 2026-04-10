#!/bin/sh

# Enable FIPS
if echo "${1}" | grep -q "sles"; then
    echo "ENABLING FIPS IN SLES SYSTEM"
    sysctl -a | grep fips
    zypper -n in -t pattern fips
    sed -i 's/^GRUB_CMDLINE_LINUX_DEFAULT="/&fips=1 /'  /etc/default/grub
    grub2-mkconfig -o /boot/grub2/grub.cfg
    mkinitrd
elif echo "${1}" | grep -Eq "rocky|centos|rhel"; then
    echo "ENABLING FIPS IN RPM-BASED SYSTEM"
    sysctl -a | grep fips
    fips-mode-setup --enable
    if [ ! -r /proc/sys/crypto/fips_enabled ] || [ "$(cat /proc/sys/crypto/fips_enabled)" != "1" ]; then
        echo "FIPS has been configured but is not active yet. A reboot is required before the image build can complete." >&2
        exit 1
    fi
    sed -i 's/\(PubkeyAcceptedKeyTypes=\)/\1ssh-rsa,/' /etc/crypto-policies/back-ends/opensshserver.config
fi 
