#!/bin/sh

while getopts ldtgo:p:k:f:c:v:h option
do 
    case "${option}"
        in
        l) log="debug";;
        d) action="deploy";;
        t) action="terminate";;
        g) action="get_running";;
        o) osname=${OPTARG};;
        p) prefix=${OPTARG};;
        k) key_name=${OPTARG};;
        f) pem_file_path=${OPTARG};;
        c) count=${OPTARG};;
        v) volume_size=${OPTARG};;
        ?|h)
            echo "
        Usage: 
            
            $(basename $0) [-l] [-d] [-t] [-g] [-o osname] [-p prefix] [-k key_name] [-f pem_file_path] [-c count] [-v volume_size] [-h]

            -l: logging is in 'debug' mode and detailed
            -d: deploy ec2 instances. displays ssh command output to setup deployed. 
            -t: terminate ec2 instances
            -g: get_running ec2 instances
            only one operation will be performed at one test run: deploy | terminate | get_running - if you provide all, the last action get_running overrides.
            -o osname: Format: osnameVersion_architecture. architecture specified only for 'arm'. default is x86
            Ex:
                RHEL: rhel9_arm, rhel9, rhel9.1_arm, rhel9.1, rhel9.2_arm, rhel9.2 
                      rhel8.8, rhel8.7, rhel8.7_arm, rhel8.6, rhel8.6_arm
                ** rhel x86 versions are packer generated modified ami's with enable fips/disable ntwk mgmt; arm versions are unedited
                *** Did not find rhel8.8_arm ami
                SLES: sles15sp4_arm, sles15sp4
                Ubuntu: ubuntu22.4, ubuntu22.4_arm, ubuntu20.4, ubuntu20.4_arm
                Oracle Linux: OL8.6, OL8.7, OL8.8(ProComputer), OL9, OL9.1, OL9.2
                **  All are packer generated AMIs
                    Most images are packer edited from Tiov IT - use 'cloud-user' for ssh
                    AMI packer generated from ProComputer - use 'ec2-user' for ssh. Double check the firewall service.
                *** Did not find arm ami's for Oracle Linux
                Rocky: rocky8.6, rocky8.6_arm, rocky8.7(packer edited), rocky8.7_arm, rock8.8, rocky8.8_arm, rocky9, rocky9.1, rocky9.1_arm, rocky9.2, rocky9.2_arm
            -p prefix: used to append to name tag the ec2 instance - you can also export PREFIX var to set as default value, if not using this option
            -k key_name: key-pair login name used from aws registry to login securely to your ec2 instances - export KEY_NAME var to set as default value, if not using this option
            -f pem_file_path: absolute file path of your .pem file - for ssh command to your ec2 instances - export PEM_FILE_PATH var to set as default value, if not using this option
            -c count: How many ec2 instances do you want to launch?
            -v volume_size: Recommend 20 (20GB for EBS volume) for k3s setup. Recommend 30 (30GB for EBS volume)for rke2 setups. Default value is 30.
            -h help - usage is displayed
            "
            exit 1
            ;;
    esac
done

if [[ $log == "debug" ]]; then
    echo "action = $action"
    echo "osname = $osname"
    echo "prefix = $prefix"
    echo "key_name = $key_name"
    echo "pem_file_path = $pem_file_path"
    echo "count = $count"
    echo "EBS volume size = $volume_size"
fi

public_dns_file_path="$PWD/public_dns"
instances_file_path="$PWD/instances"
deployed_file_path="$PWD/deployed"
terminate_file_path="$PWD/terminate"

# Which OS to deploy
# Note: the x86 image ami for rhel and OL versions are packer generated enabling fips and disabling network management setup
# The source_image_id from which the new packer generated ami (image_id) was created is also noted in the case
# error occurs for the arm versions as of now. So they are regular ami's without having run the enable fips/disable nm etc.
case $osname in
# RHEL
    rhel9.2)
        image_id="ami-018682060296f1acb"  # Packer generated ami running ami_rhel.json on the source_image_id in us-east-2 region
        source_image_id="ami-02b8534ff4b424939"
        ;;
    rhel9.2_arm) image_id="ami-06df7225cc50ee1a3";;
    rhel9.1)
        image_id="ami-07045b44fb937da35"  # Packer generated ami running ami_rhel.json on the source_image_id in us-east-2 region
        source_image_id="ami-0beb7639ce29e0148"
        ;;
    rhel9.1_arm) image_id="ami-0702921e4ba107be7";;
    rhel9)
        image_id="ami-00f8614ef399e8619"  # Packer generated ami running ami_rhel.json on the source_image_id in us-east-2 region
        source_image_id="ami-0b8384a301b67118e"
        ;;
    rhel9_arm) image_id="ami-0a33bf6de464f0857";;
    rhel8.8)
        image_id="ami-01b9a0d844d27c780"  # Packer generated ami running ami_rhel.json on the source_image_id in us-east-2 region
        source_image_id="ami-064360afd86576543"
        ;;
    rhel8.8_arm) echo "Did not find an ami for this. Exiting." ; exit;;
    rhel8.7) 
        image_id="ami-0defbb5087b2b63c1"  # Packer generated pre-existing image
        source_image_id="Pre-existing packer image"
        ;;
    rhel8.7_arm) image_id="ami-0ea3f28d61bb27a55";;
    rhel8.6)
        image_id="ami-0bb95ea8da9bc48e0"   # Packer generated pre-existing image
        source_image_id="Pre-existing packer image"
        ;;
    rhel8.6_arm) image_id="ami-0967b839a12c34f06";;
# SLES
    sles15sp4) image_id="ami-0fb3a91b7ce257ec1";;
    sles15sp4_arm) image_id="ami-052fd3067d337faf6";;
# Ubuntu
    ubuntu22.4) image_id="ami-024e6efaf93d85776";;
    ubuntu22.4_arm) image_id="ami-08fdd91d87f63bb09";;
    ubuntu20.4) image_id="ami-0430580de6244e02e";;
    ubuntu20.4_arm) image_id="ami-0071e4b30f26879e2";;
# Oracle Linux
# Note: The AMIs from Tiov IT use 'cloud-user' as the ssh username
# AMIs from ProComputer user ssh user 'ec2-user'. The packer generated from this AMI says: firewalld.service could not be found
# TODO need to see what firewall service ProComputer AMIs are using, and fix them.
# 8.8 version alone is from ProComputer. The rest are from Tiov IT source image AMI
# Could not find ARM version AMIs for Oracle Linux.
    OL8.6) 
        image_id="ami-02044a75f9562cb63"
        source_image_id="ami-0f2049f50900caa90"
        ;;
    OL8.6_arm) echo "Did not find an ami for this. Exiting." ; exit;;
    OL8.7)
        image_id="ami-054a49e0c0c7fce5c"  # Packer generated pre-existing ami
        source_image_id="Pre existing packer image"
        ;;
    OL8.7_arm) echo "Did not find an ami for this. Exiting." ; exit;;
    OL8.8)
        image_id="ami-0cd326a634acebf17"  # packer generated from ProComputer. packer log: firewalld.service could not be found 
        source_image_id="ami-012cc9c259aba3097"
        ssh_user="ec2-user"
        ;;
    OL8.8_arm) echo "Did not find an ami for this. Exiting." ; exit;;
    OL9)
        image_id="ami-0aa0d55ea757321b4"
        source_image_id="ami-0b04c8dbeb20a9d8c"
        # image_id="ami-0787a0db6f5308066"  # packer generated from ProComputer. packer log: firewalld.service could not be found
        # source_image_id="ami-0320e81c16ae2d4d4"
        # ssh_user="ec2-user"
        ;;
    OL9_arm) echo "Did not find an ami for this. Exiting." ; exit;;
    OL9.1)
        image_id="ami-01680e3ddcae57326"
        source_image_id="ami-045f72873d59547dc"
        # image_id="ami-0e572f06bbf6a0049"  # packer generated from ProComputer. packer log: firewalld.service could not be found
        # source_image_id="ami-0ca33870ec73abf78"
        # ssh_user="ec2-user"
        ;;
    OL9.1_arm) echo "Did not find an ami for this. Exiting." ; exit;;
    OL9.2)
        image_id="ami-0c50bf6c5b057201a"
        source_image_id="ami-007822fffce54749b"
        # image_id="ami-0debd32745f38204f"  # packer generated from ProComputer. packer log: firewalld.service could not be found
        # source_image_id="ami-0fee377e2c84d751b"
        # ssh_user="ec2-user"
        ;;
    OL9.2_arm) echo "Did not find an ami for this. Exiting." ; exit;;
# Rocky Linux
    rocky8.6) image_id="ami-0072f50382eb71b1d";;
    rocky8.6_arm) image_id="ami-0c92c0d181d0cfa1e";;
    rocky8.7)
        image_id="ami-05ab2eb74c93eb441"  # Packer generated pre-existing ami
        source_image_id="Pre existing packer image"
        ;;
    rocky8.7_arm) image_id="ami-0491a50679ee1bc89";;
    rocky8.8) image_id="ami-0425d70f0df70df0e";;
    rocky8.8_arm) image_id="ami-074e816b93be89812";;
    rocky9) image_id="ami-05d9eb66565e1792c";;
    rocky9_arm) echo "Did not find an ami for this. Exiting." ; exit;;
    rocky9.1) image_id="ami-01778de3d921acbe9";;
    rocky9.1_arm) image_id="ami-0fd23c283a54fb00d";;
    rocky9.2) image_id="ami-0140491b434cb5296";;
    rocky9.2_arm) image_id="ami-03a4cf1ef87c11545";;
# Default value when any other os name was provided or was empty
    *)
        if [[ $osname ]]; then
            echo "FATAL: Wrong OS Name. Please use -h to get usage info"
            exit
        else
            if [[ $action == "deploy" || $action == "get_running" ]]; then
                osname="ubuntu22.4"
                if [[ $log == "debug" ]]; then
                    echo "WARN: Setting osname = $osname"
                fi
                image_id="ami-024e6efaf93d85776"
                # image_id="ami-0a695f0d95cefc163"  # previous image id used
                ssh_user="ubuntu"
                instance_type="t3.medium"
            else
                # Terminates 'all' instances irrespective of os - if not provided as a cmd line argument
                osname="all"
                echo "INFO: Going to terminate $osname running ec2 instances"
            fi
        fi
        ;;
esac

if [[ -z $osname ]]; then
    # Default is ubuntu 22.04 for deploy/get_running actions
    if [[ $action == "deploy" || $action == "get_running" ]]; then
        osname="ubuntu22.4"
        if [[ $log == "debug" ]]; then
            echo "WARN: Setting osname = $osname"
        fi
        image_id="ami-0a695f0d95cefc163"
        ssh_user="ubuntu"
        instance_type="t3.medium"
    else
        # Terminates 'all' instances irrespective of os - if not provided as a cmd line argument
        osname="all"
        echo "INFO: Going to terminate $osname running ec2 instances"
    fi
fi

# Set SSH User based on os and if ssh_user was not already assigned already

if [[ $osname == *"ubuntu"* && -z $ssh_user ]]; then
    ssh_user="ubuntu"
fi
if [[ $osname == *"rocky"* && -z $ssh_user ]]; then
    ssh_user="rocky"
fi
if [[ $osname == *"OL"* && -z $ssh_user ]]; then
    # Tiov IT AMIs and Packer AMIs generated from them use 'cloud-user'
    # ProComputer AMIs user 'ec2-user' as the ssh username
    ssh_user="cloud-user"
fi
if [[ -z $ssh_user ]]; then
    ssh_user="ec2-user"
fi
if [[ $log == "debug" ]]; then
    echo "ssh_user = $ssh_user"
    echo "INFO: If ssh user did not work: SSH User Alternatives: 'ec2-user' or 'root' or 'cloud-user' ; rocky linux user is: 'rocky'; ubuntu user is: 'ubuntu'"
fi

# Set instance_type based on os architecture

if [[ $osname == *"arm"* ]]; then
    instance_type="a1.large"
else
    instance_type="t3.medium"
fi
if [[ $log == "debug" ]]; then
    echo "instance_type = $instance_type"
fi
# Notify if the ami is packer generated during deploy
if [[ $action == "deploy" ]]; then
    if [[ -z $source_image_id ]]; then
        echo "INFO: Unedited(non-packer) AMI used for $osname"
        if [[ $osname == *"rhel"* ]]; then
            echo "     Kindly edit enable fips and disable network mgmt as needed manually"
        fi
        if [[ $osname == *"OL"* ]]; then
            echo "      Please run disable firewall/modify user_data steps manually"
        fi
    else
        if [[ $source_image_id == *"ami"* ]]; then
            echo "INFO: This AMI is packer generated from $source_image_id"
            if [[ $osname == *"rhel"* || $osname == *"rocky"* ]]; then
                echo "      Enable FIPS and Disable Network Management has been pre-run in the AMI for you"
            fi
            if [[ $osname == *"OL"* ]]; then
                echo "      Disable Firewall and Modify user_data has been pre-run in the AMI for you"
            fi
        else
            echo "Using $source_image_id"
        fi
    fi
fi

# Set some default values if they were not provided as args

if [[ $action == "deploy" ]]; then
    if [[ -z $count ]]; then
        # Default count value to 4 ec2 instances
        count=4
        if [[ $log == "debug" ]]; then
            echo "WARN: Setting count = $count"
        fi
    fi

    if [[ -z $volume_size ]]; then
        # Default volume_size to 30G for RKE2 setup
        volume_size=30
        if [[ $log == "debug" ]]; then
            echo "WARN: Setting volume_size = $volume_size"
        fi
    fi
    if [[ -z $prefix ]]; then
        if [[ -z $PREFIX ]]; then
            echo "FATAL: Either use -p option to set prefix or set env var/ export PREFIX to skip this option. Cannot proceed with deploy action without this. Exiting the script."
            exit 1
        fi
        if [[ $log == "debug" ]]; then
            echo "WARN: Setting prefix = $PREFIX  -> Environment var PREFIX value"
        fi
        prefix=$PREFIX
    fi
fi


if [[ -z $key_name ]]; then
    if [[ -z $KEY_NAME ]]; then
        echo "FATAL: Either set -k value for key_name or set env var/export variable KEY_NAME to skip this option. Cannot proceed without this value. Exiting the script."
        exit 1
    fi
    if [[ $log == "debug" ]]; then
        echo "WARN: Setting key_name = $KEY_NAME -> Environement var KEY_NAME value"
    fi
    key_name=$KEY_NAME
fi

if [[ $action == "deploy" || $action == "get_running" ]]; then
    if [[ -z $pem_file_path ]]; then
        if [[ -z $PEM_FILE_PATH ]]; then
            echo "FATAL: Either set -f value for pem_file_path or set env var/export PEM_FILE_PATH to skip this option. Cannot proceed without this value. Exiting the script."
            exit 1
        fi
        if [[ $log == "debug" ]]; then
            echo "WARN: Setting pem_file_path = $PEM_FILE_PATH -> Environment var PEM_FILE_PATH value"
        fi
        pem_file_path=$PEM_FILE_PATH
    fi
fi

echo "*************************
ACTION STAGE: $action
*************************"

case $action in
    deploy)
        echo "Deploying OS: $osname ImageID: $image_id SSH_User: $ssh_user" 
        aws ec2 run-instances --image-id $image_id --instance-type $instance_type --count $count --key-name $key_name --security-group-ids sg-0e753fd5550206e55 --block-device-mappings "[{\"DeviceName\":\"/dev/sda1\",\"Ebs\":{\"VolumeSize\":$volume_size,\"DeleteOnTermination\":true}}]" --tag-specifications "[{\"ResourceType\": \"instance\", \"Tags\": [{\"Key\": \"Name\", \"Value\": \"$prefix-$osname\"}]}]" > /dev/null
        # sleep 30  # To ensure the system is actually running by the time we use the ssh command output by this script.
        aws ec2 describe-instances --filters Name=key-name,Values=$key_name Name=image-id,Values=$image_id Name=instance-state-name,Values="running" > $deployed_file_path
        grep PublicDns $deployed_file_path | grep -v "''" | awk '{print $2}' | uniq > $public_dns_file_path
        while read -r line
        do
            echo "ssh -i \"$pem_file_path\" $ssh_user@$line"
        done < $public_dns_file_path
        rm $public_dns_file_path $deployed_file_path
        ;;
    get_running)
        # Running instance ssh cmd is output
        echo "Getting setups for OS: $osname with ImageID: $image_id and SSH_User: $ssh_user"
        aws ec2 describe-instances --filters Name=key-name,Values=$key_name Name=image-id,Values=$image_id Name=instance-state-name,Values="running" > $deployed_file_path
        grep PublicDns $deployed_file_path | grep -v "''" | awk '{print $2}' | uniq > $public_dns_file_path
        while read -r line
        do
            echo "ssh -i \"$pem_file_path\" $ssh_user@$line"
        done < $public_dns_file_path
        rm $public_dns_file_path $deployed_file_path
        ;;
    terminate)
        if [[ $osname == "all" ]]; then
            echo "Initiate termination for all running instances"
            aws ec2 describe-instances --filters Name=key-name,Values=$key_name Name=instance-state-name,Values="running" > $terminate_file_path
        else
            echo "Initiating termination for running ec2 instances with OS: $osname with ImageID: $image_id"
            aws ec2 describe-instances --filters Name=key-name,Values=$key_name Name=image-id,Values=$image_id Name=instance-state-name,Values="running" > $terminate_file_path
        fi
        grep InstanceId $terminate_file_path | awk '{print $2}' > $instances_file_path
        while read -r line
        do
            echo "Terminating instance id: $line"
            aws ec2 terminate-instances --instance-ids $line > /dev/null
        done < $instances_file_path
        rm $instances_file_path $terminate_file_path
        ;;
esac