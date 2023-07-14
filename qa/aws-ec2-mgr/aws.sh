#!/bin/sh

while getopts ldtgo:p:k:f:c:v:s:h OPTION
do 
    case "${OPTION}"
        in
        l) LOG="debug";;
        d) ACTION="deploy";;
        t) ACTION="terminate";;
        g) ACTION="get_running";;
        o) OS_NAME=${OPTARG};;
        p) PREFIX_TAG=${OPTARG};;
        k) KEY_NAME_VAR=${OPTARG};;
        f) PEM_FILE_PATH_VAR=${OPTARG};;
        c) COUNT=${OPTARG};;
        v) VOLUME_SIZE=${OPTARG};;
        s) SERVER_COUNT=${OPTARG};;
        h|?)
            echo "
        Usage: 
            
            $(basename "$0") [-l] [-d] [-t] [-g] [-o os_name] [-p prefix] [-k key_name] [-f pem_file_path] [-c count] [-v volume_size] [-h]

            -l: logging is in 'debug' mode and detailed
            -d: deploy ec2 instances. displays ssh command output to setup deployed. 
            -t: terminate ec2 instances
            -g: get_running ec2 instances
            only one operation will be performed at one test run: deploy | terminate | get_running - if you provide all, the last action get_running overrides.
            -o os_name: Format: {os_name}{version}_{architecture}. architecture specified only for 'arm'. default is x86
            Ex:
                RHEL: rhel9_arm, rhel9, rhel9.1_arm, rhel9.1, rhel9.2_arm, rhel9.2 
                      rhel8.8, rhel8.7, rhel8.7_arm, rhel8.6, rhel8.6_arm
                ** rhel x86 versions are packer generated modified ami's with enable fips/disable ntwk mgmt; arm versions are unedited
                *** Did not find rhel8.8_arm ami
                SLES: sles15sp4_arm, sles15sp4
                Ubuntu: ubuntu22.4, ubuntu22.4_arm, ubuntu20.4, ubuntu20.4_arm
                Oracle Linux: OL8.6, OL8.7, OL8.8 (ProComputer), OL9, OL9.1, OL9.2
                **  All are packer generated AMIs
                    Most images are packer edited from Tiov IT - use 'cloud-user' for ssh
                    AMI packer generated from ProComputer - use 'ec2-user' for ssh. Double check the firewall service.
                *** Did not find arm ami's for Oracle Linux
                Rocky: rocky8.6, rocky8.6_arm, rocky8.7 (packer edited), rocky8.7_arm, rock8.8, rocky8.8_arm, rocky9, rocky9.1, rocky9.1_arm, rocky9.2, rocky9.2_arm
            -p prefix: used to append to name tag the ec2 instance - you can also export PREFIX var to set as default value, if not using this option
            -k key_name: key-pair login name used from aws registry to login securely to your ec2 instances - export KEY_NAME var to set as default value, if not using this option
            -f pem_file_path: absolute file path of your .pem file - for ssh command to your ec2 instances - export PEM_FILE_PATH var to set as default value, if not using this option
            -c count: How many ec2 instances do you want to launch?
            -v volume_size: Recommend 20 (20GB for EBS volume) for k3s setup. Recommend 30 (30GB for EBS volume)for rke2 setups. Default value is 30.
            -s server_count: Can be 3 for 3 servers 1 agent or 2 for 2 servers and 2 agents; To be used with the -g get_running option or -d deploy option
            -h help - usage is displayed
            "
            exit 1
            ;;
    esac
done

if [ "${LOG}" = "debug" ]; then
    echo "ACTION = ${ACTION}"
    echo "OS_NAME = ${OS_NAME}"
    echo "PREFIX_TAG = ${PREFIX_TAG}"
    echo "KEY_NAME_VAR = ${KEY_NAME_VAR}"
    echo "PEM_FILE_PATH_VAR = ${PEM_FILE_PATH_VAR}"
    echo "COUNT = ${COUNT}"
    echo "EBS VOLUME_SIZE = ${VOLUME_SIZE}"
fi

PUBLIC_IPS_FILE_PATH="${PWD}/public_ips"
INSTANCES_FILE_PATH="${PWD}/instances"
DEPLOYED_FILE_PATH="${PWD}/deployed"
TERMINATE_FILE_PATH="${PWD}/terminate"

# Which OS to deploy
# Note: the x86 image ami for rhel and OL versions are packer generated enabling fips and disabling network management setup
# The SOURCE_IMAGE_ID from which the new packer generated ami (IMAGE_ID) was created is also noted in the case
# error occurs for the arm versions as of now. So they are regular ami's without having run the enable fips/disable nm etc.
case ${OS_NAME} in
# RHEL
    rhel9.2)

        # IMAGE_ID="ami-024fe43f3e88e89b5"
        # SOURCE_IMAGE_ID="ami-00bc24f98893a4bef"
        IMAGE_ID="ami-082bf7cc12db545b9"  # Enable FIPS, Disable NtwkMgr, Disable firewalld.service
        # IMAGE_ID="ami-018682060296f1acb"  # Packer generated ami running ami_rhel.json on the SOURCE_IMAGE_ID in us-east-2 region
        SOURCE_IMAGE_ID="ami-02b8534ff4b424939"
        ;;
    rhel9.2_arm)
        IMAGE_ID="ami-08e34b72961f79fae"  # Enable FIPS, Disable NtwkMgr, Disable firewalld.service
        SOURCE_IMAGE_ID="ami-06df7225cc50ee1a3"
        ;;
    rhel9.1)
        IMAGE_ID="ami-04c5820302c03d3ba"
        SOURCE_IMAGE_ID="ami-045f72873d59547dc"
        SSH_USER="cloud-user"
        # IMAGE_ID="ami-07045b44fb937da35"  # Packer generated ami running ami_rhel.json on the SOURCE_IMAGE_ID in us-east-2 region
        # SOURCE_IMAGE_ID="ami-0beb7639ce29e0148"
        ;;
    rhel9.1_arm) 
        IMAGE_ID="ami-022b0843d9926c2ea"
        SOURCE_IMAGE_ID="ami-0702921e4ba107be7"
        ;;
    rhel9)
        IMAGE_ID="ami-00f8614ef399e8619"  # Packer generated ami running ami_rhel.json on the SOURCE_IMAGE_ID in us-east-2 region
        SOURCE_IMAGE_ID="ami-0b8384a301b67118e"
        ;;
    rhel9_arm)
        IMAGE_ID="ami-0ab8d8846ed3b5a7d"
        SOURCE_IMAGE_ID="ami-0a33bf6de464f0857"
        ;;
    rhel8.8)
        IMAGE_ID="ami-01b9a0d844d27c780"  # Packer generated ami running ami_rhel.json on the SOURCE_IMAGE_ID in us-east-2 region
        SOURCE_IMAGE_ID="ami-064360afd86576543"
        ;;
    rhel8.8_arm) echo "Did not find an ami for this. Exiting." ; exit;;
    rhel8.7) 
        IMAGE_ID="ami-0defbb5087b2b63c1"  # Packer generated pre-existing image
        SOURCE_IMAGE_ID="Pre-existing packer image"
        ;;
    rhel8.7_arm)
        IMAGE_ID="ami-0085ea62fc18e0154" 
        SOURCE_IMAGE_ID="ami-0ea3f28d61bb27a55"
        ;;
    rhel8.6)
        IMAGE_ID="ami-0bb95ea8da9bc48e0"   # Packer generated pre-existing image
        SOURCE_IMAGE_ID="Pre-existing packer image"
        ;;
    rhel8.6_arm) 
        IMAGE_ID="ami-0f9fc13cd1cfaab88"
        SOURCE_IMAGE_ID="ami-0967b839a12c34f06"
        ;;
# SLES
    sles15sp4) IMAGE_ID="ami-0fb3a91b7ce257ec1";;
    sles15sp4_arm) IMAGE_ID="ami-052fd3067d337faf6";;
# Ubuntu
    ubuntu22.4) IMAGE_ID="ami-024e6efaf93d85776";;
    ubuntu22.4_arm) IMAGE_ID="ami-08fdd91d87f63bb09";;
    ubuntu20.4) IMAGE_ID="ami-0430580de6244e02e";;
    ubuntu20.4_arm) IMAGE_ID="ami-0071e4b30f26879e2";;
# Oracle Linux
# Note: The AMIs from Tiov IT use 'cloud-user' as the ssh username
# AMIs from ProComputer user ssh user 'ec2-user'. The packer generated from this AMI says: firewalld.service could not be found
# TODO need to see what firewall service ProComputer AMIs are using, and fix them.
# 8.8 version alone is from ProComputer. The rest are from Tiov IT source image AMI
# Could not find ARM version AMIs for Oracle Linux.
    OL8.6) 
        IMAGE_ID="ami-02044a75f9562cb63"
        SOURCE_IMAGE_ID="ami-0f2049f50900caa90"
        ;;
    OL8.6_arm) echo "Did not find an ami for this. Exiting." ; exit;;
    OL8.7)
        IMAGE_ID="ami-054a49e0c0c7fce5c"  # Packer generated pre-existing ami
        SOURCE_IMAGE_ID="Pre existing packer image"
        ;;
    OL8.7_arm) echo "Did not find an ami for this. Exiting." ; exit;;
    OL8.8)
        IMAGE_ID="ami-0e17d020894274cb7" # Disable firewall, UserGrowPart, Disable NtwkMgr were run
        # IMAGE_ID="ami-0cd326a634acebf17"  # packer generated from ProComputer. packer log: firewalld.service could not be found 
        SOURCE_IMAGE_ID="ami-012cc9c259aba3097"
        SSH_USER="ec2-user"
        ;;
    OL8.8_arm) echo "Did not find an ami for this. Exiting." ; exit;;
    OL9)
        IMAGE_ID="ami-0aa0d55ea757321b4"
        SOURCE_IMAGE_ID="ami-0b04c8dbeb20a9d8c"
        # IMAGE_ID="ami-0787a0db6f5308066"  # packer generated from ProComputer. packer log: firewalld.service could not be found
        # SOURCE_IMAGE_ID="ami-0320e81c16ae2d4d4"
        # SSH_USER="ec2-user"
        ;;
    OL9_arm) echo "Did not find an ami for this. Exiting." ; exit;;
    OL9.1)
        IMAGE_ID="ami-01680e3ddcae57326"
        SOURCE_IMAGE_ID="ami-045f72873d59547dc"
        # IMAGE_ID="ami-0e572f06bbf6a0049"  # packer generated from ProComputer. packer log: firewalld.service could not be found
        # SOURCE_IMAGE_ID="ami-0ca33870ec73abf78"
        # SSH_USER="ec2-user"
        ;;
    OL9.1_arm) echo "Did not find an ami for this. Exiting." ; exit;;
    OL9.2)
        IMAGE_ID="ami-0d77b6b12ba00534b" # Esteban validated with this ami
        # IMAGE_ID="ami-0c50bf6c5b057201a"
        # SOURCE_IMAGE_ID="ami-007822fffce54749b"
        # IMAGE_ID="ami-0debd32745f38204f"  # packer generated from ProComputer. packer log: firewalld.service could not be found
        # SOURCE_IMAGE_ID="ami-0fee377e2c84d751b"
        # SSH_USER="ec2-user"
        ;;
    OL9.2_arm) echo "Did not find an ami for this. Exiting." ; exit;;
# Rocky Linux
    rocky8.6) IMAGE_ID="ami-0072f50382eb71b1d";;
    rocky8.6_arm) IMAGE_ID="ami-0c92c0d181d0cfa1e";;
    rocky8.7)
        IMAGE_ID="ami-05ab2eb74c93eb441"  # Packer generated pre-existing ami
        SOURCE_IMAGE_ID="Pre existing packer image"
        ;;
    rocky8.7_arm) IMAGE_ID="ami-0491a50679ee1bc89";;
    rocky8.8) IMAGE_ID="ami-0425d70f0df70df0e";;
    rocky8.8_arm) IMAGE_ID="ami-074e816b93be89812";;
    rocky9) IMAGE_ID="ami-05d9eb66565e1792c";;
    rocky9_arm) echo "Did not find an ami for this. Exiting." ; exit;;
    rocky9.1) IMAGE_ID="ami-01778de3d921acbe9";;
    rocky9.1_arm) IMAGE_ID="ami-0fd23c283a54fb00d";;
    rocky9.2) IMAGE_ID="ami-0140491b434cb5296";;
    rocky9.2_arm) IMAGE_ID="ami-03a4cf1ef87c11545";;
# Default value when any other os name was provided or was empty
    *)
        if [ "${OS_NAME}" ]; then
            echo "FATAL: Wrong OS Name. Please use -h to get usage info"
            exit
        else
            if [ "${ACTION}" = "deploy" ] || [ "${ACTION}" = "get_running" ]; then
                OS_NAME="ubuntu22.4"
                if [ "${LOG}" = "debug" ]; then
                    echo "WARN: Setting OS_NAME = ${OS_NAME}"
                fi
                IMAGE_ID="ami-024e6efaf93d85776" # AWS EC2 console picks this latest x86 ami 
                # IMAGE_ID="ami-097a2df4ac947655f" # RKE2 jenkins job uses this ami: https://jenkins.int.rancher.io/job/rke2-tests/view/cluster-creation/job/rke2_freeform_create_and_validate/build?delay=0sec
                # IMAGE_ID="ami-0283a57753b18025b" # K3S jenkins job uses this ami: https://jenkins.int.rancher.io/job/rancher_qa/view/k3s/job/create_k3s_ha_cluster/build?delay=0sec
                # IMAGE_ID="ami-0a695f0d95cefc163"  # previous image id used
                SSH_USER="ubuntu"
                INSTANCE_TYPE="t3.medium"
            else
                # Terminates 'all' instances irrespective of os - if not provided as a cmd line argument
                OS_NAME="all"
                echo "INFO: Going to terminate ${OS_NAME} running ec2 instances"
            fi
        fi
        ;;
esac

# Set SSH User based on os and if SSH_USER was not already assigned already
if [ -z "${SSH_USER}" ]; then
    if echo "${OS_NAME}" | grep -q "ubuntu" ; then
        SSH_USER="ubuntu"
    fi
    if echo "${OS_NAME}" | grep -q "rocky"; then
        SSH_USER="rocky"
    fi
    if echo "${OS_NAME}" | grep -q "OL"; then
        # Tiov IT AMIs and Packer AMIs generated from them use 'cloud-user'
        # ProComputer AMIs user 'ec2-user' as the ssh username
        SSH_USER="cloud-user"
    fi
    if [ -z "${SSH_USER}" ]; then
        SSH_USER="ec2-user"
    fi
fi
if [ "${LOG}" = "debug" ]; then
    echo "SSH_USER = ${SSH_USER}"
    echo "INFO: If ssh user did not work: SSH User Alternatives: 'ec2-user' or 'root' or 'cloud-user' ; rocky linux user is: 'rocky'; ubuntu user is: 'ubuntu'"
fi

# Set instance_type based on os architecture

if echo "${OS_NAME}" | grep -q "arm"; then
    INSTANCE_TYPE="a1.large"
else
    INSTANCE_TYPE="t3.medium"
fi
if [ "${LOG}" = "debug" ]; then
    echo "INSTANCE_TYPE = ${INSTANCE_TYPE}"
fi
# Notify if the ami is packer generated during deploy
if [ "${ACTION}" = "deploy" ]; then
    if [ -z "${SOURCE_IMAGE_ID}" ]; then
        echo "INFO: Unedited(non-packer) AMI used for ${OS_NAME}"
        if echo "${OS_NAME}" | grep -q "rhel"; then
            echo "     Kindly edit enable fips and disable network mgmt as needed manually"
        fi
        if echo "${OS_NAME}" | grep -q "OL"; then
            echo "      Please run disable firewall/modify user_data steps manually"
        fi
    else
        if echo "${SOURCE_IMAGE_ID}" | grep -q "ami"; then
            echo "INFO: This AMI is packer generated from ${SOURCE_IMAGE_ID}"
            if echo "${OS_NAME}" | grep -q "rhel" || echo "${OS_NAME}" | grep -q "rocky"; then
                echo "      Enable FIPS and Disable Network Management has been pre-run in the AMI for you"
            fi
            if echo "${OS_NAME}" | grep -q "OL"; then
                echo "      Disable Firewall and Modify user_data has been pre-run in the AMI for you"
            fi
        else
            echo "Using ${SOURCE_IMAGE_ID}"
        fi
    fi
fi

# Set some default values if they were not provided as args

if [ "${ACTION}" = "deploy" ] || [ "${ACTION}" = "get_running" ]; then
    if [ -z "${COUNT}" ]; then
        # Default count value to 4 ec2 instances
        COUNT=4
        if [ "${LOG}" = "debug" ]; then
            echo "WARN: Setting COUNT = ${COUNT}"
        fi
    fi

    if [ -z "${VOLUME_SIZE}" ]; then
        # Default volume_size to 30G for RKE2 setup
        VOLUME_SIZE=30
        if [ "${LOG}" = "debug" ]; then
            echo "WARN: Setting VOLUME_SIZE = ${VOLUME_SIZE}"
        fi
    fi
    if [ -z "${PREFIX_TAG}" ]; then
         if [ -z "${PREFIX}" ]; then
            echo "FATAL: Either use -p option to set prefix or set env var/ export PREFIX to skip this option. Cannot proceed with deploy action without this. Exiting the script."
            exit 1
        fi
        if [ "${LOG}" = "debug" ]; then
            echo "WARN: Setting PREFIX_TAG = ${PREFIX}  -> Environment var PREFIX value"
        fi
        PREFIX_TAG=${PREFIX}
    fi
fi


if [ -z "${KEY_NAME_VAR}" ]; then
    if [ -z "$KEY_NAME" ]; then
        echo "FATAL: Either set -k value for key_name or set env var/export variable KEY_NAME to skip this option. Cannot proceed without this value. Exiting the script."
        exit 1
    fi
    if [ "${LOG}" = "debug" ]; then
        echo "WARN: Setting KEY_NAME_VAR = $KEY_NAME -> Environement var KEY_NAME value"
    fi
    KEY_NAME_VAR=$KEY_NAME
fi

if [ "${ACTION}" = "deploy" ] || [ "${ACTION}" = "get_running" ]; then
    if [ -z "${PEM_FILE_PATH_VAR}" ]; then
        if [ -z "$PEM_FILE_PATH" ]; then
            echo "FATAL: Either set -f value for pem_file_path or set env var/export PEM_FILE_PATH to skip this option. Cannot proceed without this value. Exiting the script."
            exit 1
        fi
        if [ "${LOG}" = "debug" ]; then
            echo "WARN: Setting PEM_FILE_PATH_VAR = $PEM_FILE_PATH -> Environment var PEM_FILE_PATH value"
        fi
        PEM_FILE_PATH_VAR=$PEM_FILE_PATH
    fi
fi

if [ -z "${SERVER_COUNT}" ]; then
    SERVER_COUNT=3
fi

if [ "${SERVER_COUNT}" ]; then
    AGENT_COUNT=$((COUNT-SERVER_COUNT))
fi


get_ips () {
    aws ec2 describe-instances --filters Name=key-name,Values="${KEY_NAME_VAR}" Name=image-id,Values="${IMAGE_ID}" Name=instance-state-name,Values="running" > "${DEPLOYED_FILE_PATH}"
    grep PublicIp "${DEPLOYED_FILE_PATH}" | grep -v "''" | awk '{print $2}' | uniq > "${PUBLIC_IPS_FILE_PATH}"
}

get_ssh_info () {
    while read -r LINE
    do
        echo "ssh -i \"${PEM_FILE_PATH_VAR}\" ${SSH_USER}@${LINE}"
    done < "${PUBLIC_IPS_FILE_PATH}"    
}

get_setup_vars () {
    while read -r LINE
    do
        if [ "${SERVER_COUNT}" = 0 ]; then
            echo "AGENT${AGENT_COUNT}=\"${LINE}\""
            AGENT_COUNT=$((AGENT_COUNT-1))
        else
            echo "SERVER${SERVER_COUNT}=\"${LINE}\""
            SERVER_COUNT=$((SERVER_COUNT-1))
        fi
    done < "${PUBLIC_IPS_FILE_PATH}"
}

get_setup_info () {
    get_ips
    get_ssh_info
    get_setup_vars
    rm "${PUBLIC_IPS_FILE_PATH}" "${DEPLOYED_FILE_PATH}"
}


echo "*************************
ACTION STAGE: ${ACTION}
*************************"

case "${ACTION}" in
    deploy)
        echo "Deploying OS: ${OS_NAME} ImageID: ${IMAGE_ID} SSH_USER: ${SSH_USER}" 
        aws ec2 run-instances --image-id "${IMAGE_ID}" --instance-type "${INSTANCE_TYPE}" --count "${COUNT}" --key-name "${KEY_NAME_VAR}" --security-group-ids sg-0e753fd5550206e55 --block-device-mappings "[{\"DeviceName\":\"/dev/sda1\",\"Ebs\":{\"VolumeSize\":${VOLUME_SIZE},\"DeleteOnTermination\":true}}]" --tag-specifications "[{\"ResourceType\": \"instance\", \"Tags\": [{\"Key\": \"Name\", \"Value\": \"${PREFIX_TAG}-${OS_NAME}\"}]}]" > /dev/null
        sleep 30  # To ensure the system is actually running by the time we use the ssh command output by this script.
        get_setup_info
        ;;
    get_running)
        # Running instance ssh cmd is output
        echo "Getting setups for OS: ${OS_NAME} with ImageID: ${IMAGE_ID} and SSH_USER: ${SSH_USER}"
        get_setup_info
        ;;
    terminate)
        if [ "${OS_NAME}" = "all" ]; then
            echo "Initiate termination for all running instances"
            aws ec2 describe-instances --filters Name=key-name,Values="${KEY_NAME_VAR}" Name=instance-state-name,Values="running" > "${TERMINATE_FILE_PATH}"
        else
            echo "Initiating termination for running ec2 instances with OS: ${OS_NAME} with ImageID: ${IMAGE_ID}"
            aws ec2 describe-instances --filters Name=key-name,Values="${KEY_NAME_VAR}" Name=image-id,Values="${IMAGE_ID}" Name=instance-state-name,Values="running" > "${TERMINATE_FILE_PATH}"
        fi
        grep InstanceId "${TERMINATE_FILE_PATH}" | awk '{print $2}' > "${INSTANCES_FILE_PATH}"
        while read -r LINE
        do
            echo "Terminating instance id: ${LINE}"
            aws ec2 terminate-instances --instance-ids "${LINE}" > /dev/null
        done < "${INSTANCES_FILE_PATH}"
        rm "${INSTANCES_FILE_PATH}" "${TERMINATE_FILE_PATH}"
        ;;
esac
