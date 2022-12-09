#!/bin/sh

# Enable FIPS
if [[ ${1} == *"sles"* ]]
then
  echo "ENABLING FIPS IN SLES SYSTEM"
  sysctl -a | grep fips
  zypper -n in -t pattern fips
  sed -i 's/^GRUB_CMDLINE_LINUX_DEFAULT="/&fips=1 /'  /etc/default/grub
  grub2-mkconfig -o /boot/grub2/grub.cfg
  mkinitrd
elif [[ ${1} == *"rocky"* ]] || [[ ${1} == *"centos"* ]] || [[ ${1} == *"rhel"* ]]
then
  echo "ENABLING FIPS IN RPM-BASED SYSTEM"
  sysctl -a | grep fips
  fips-mode-setup --enable
  sed -i 's/\(PubkeyAcceptedKeyTypes=\)/\1ssh-rsa,/' /etc/crypto-policies/back-ends/opensshserver.config
fi 

