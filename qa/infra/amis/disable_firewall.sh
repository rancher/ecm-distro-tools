#!/bin/sh

# Disable FirewallD
FIREWALLD_ENABLED=$(systemctl status firewalld | grep -i enabled)

if [ "${FIREWALLD_ENABLED}" ]; then 
  systemctl disable firewalld
fi
