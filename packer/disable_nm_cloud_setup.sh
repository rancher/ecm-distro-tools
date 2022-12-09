#!/bin/sh

# Disable Incompatible Services for RKE2

NM_CLOUD_SETUP_SERVICE_ENABLED=`systemctl status nm-cloud-setup.service | grep -i enabled`
NM_CLOUD_SETUP_TIMER_ENABLED=`systemctl status nm-cloud-setup.timer | grep -i enabled`

if [ "$NM_CLOUD_SETUP_SERVICE_ENABLED" ]
then 
  systemctl disable nm-cloud-setup.service
fi

if [ "$NM_CLOUD_SETUP_TIMER_ENABLED" ]
then 
  systemctl disable nm-cloud-setup.timer
fi

