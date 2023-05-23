#!/bin/sh

while getopts dtgo:p:k:f:c:v:h option
do 
    case "${option}"
        in
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
            
            $(basename $0) [-d] [-t] [-g] [-o osname] [-p prefix] [-k key_name] [-f pem_file_path] [-c count] [-v volume_size] [-h]
            
            -d: deploy ec2 instances. displays ssh command output to setup deployed. 
            -t: terminate ec2 instances
            -g: get_running ec2 instances
            only one operation will be performed at one test run: deploy | terminate | get_running - if you provide all, the last action get_running overrides.
            -o osname: Format: osnameVersion_architecture. architecture specified only for 'arm'. default is x86
            Ex:
                RHEL: rhel9_arm, rhel9, rhel9.1_arm, rhel9.1, rhel9.2_arm, rhel9.2
                SLES: sles15sp4_arm, sles15sp4
                Ubuntu: ubuntu22.4, ubuntu22.4_arm, ubuntu20.4, ubuntu20.4_arm
                Oracle Linux: OL8.7 (did not find arm version)
                Rocky: rocky8.7 (arm version seems to need optin with accepting terms https://aws.amazon.com/marketplace/pp?sku=7tvwi95pv43herd5jg0bs6cu5) 
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


echo "action: $action"
echo "osname : $osname"
echo "prefix   : $prefix"
echo "key_name : $key_name"
echo "pem_file_path   : $pem_file_path"
echo "count : $count"
echo "EBS volume size   : $volume_size"

public_dns_file_path="$PWD/public_dns"
instances_file_path="$PWD/instances"
deployed_file_path="$PWD/deployed"
terminate_file_path="$PWD/terminate"

# Which OS to deploy - the second argument to the script
case $osname in
    rhel9_arm)
        image_id="ami-0a33bf6de464f0857"
        ssh_user="ec2-user"
        instance_type="a1.large"
        ;;
    rhel9)
        image_id="ami-0b8384a301b67118e"
        ssh_user="ec2-user"
        instance_type="t3.medium"
        ;;
    rhel9.1_arm)
        image_id="ami-0702921e4ba107be7"
        ssh_user="ec2-user"
        instance_type="a1.large"
        ;;
    rhel9.1)
        image_id="ami-0beb7639ce29e0148"
        ssh_user="ec2-user"
        instance_type="t3.medium"
        ;;
    rhel9.2_arm)
        image_id="ami-06df7225cc50ee1a3"
        ssh_user="ec2-user"
        instance_type="a1.large"
        ;;
    rhel9.2)
        image_id="ami-02b8534ff4b424939"
        ssh_user="ec2-user"
        instance_type="t3.medium"
        ;;
    rhel8.7)
        image_id="ami-057094267c651958e"
        ssh_user="ec2-user"
        instance_type="t3.medium"
        ;;
    rhel8.7_arm)
        image_id="ami-0ea3f28d61bb27a55"
        ssh_user="ec2-user"
        instance_type="a1.large"
        ;;
    sles15sp4_arm)
        image_id="ami-052fd3067d337faf6"
        ssh_user="ec2-user"
        instance_type="a1.large"
        ;;
    sles15sp4)
        image_id="ami-0fb3a91b7ce257ec1"
        ssh_user="ec2-user"
        instance_type="t3.medium"
        ;;
    ubuntu22.4)
        image_id="ami-0a695f0d95cefc163"
        ssh_user="ubuntu"
        instance_type="t3.medium"
        ;;
    ubuntu22.4_arm)
        image_id="ami-0f12014c8b2f26d33"
        ssh_user="ubuntu"
        instance_type="a1.large"
        ;;
    ubuntu20.4)
        image_id="ami-06c4532923d4ba1ec"
        ssh_user="ubuntu"
        instance_type="t3.medium"
        ;;
    ubuntu20.4_arm)
        image_id="ami-090226778695b30b9"
        ssh_user="ubuntu"
        instance_type="a1.large"
        ;;
    OL8.7)
        image_id="ami-06ac6a66b683196b8"
        ssh_user="root"
        instance_type="t3.medium"
        ;;
    rocky8.7)
        image_id="ami-02fb9384e880ed67c"
        ssh_user="rocky"
        instance_type="t3.medium"
        ;;
    # rocky8.7_arm)
    #  This setup seems to require an optin 
    #     image_id="ami-0491a50679ee1bc89"
    #     ssh_user="rocky"
    #     instance_type="a1.large"
    #     ;;
esac

if [[ -z $osname ]]; then
    # Default is ubuntu 22.04 if the second argument is not provided
    if [[ $action == "deploy" || $action == "get_running" ]]; then
        echo "osname was not provided via args. Setting default value to ubuntu22.4"
        image_id="ami-0a695f0d95cefc163"
        ssh_user="ubuntu"
        instance_type="t3.medium"
        osname="ubuntu22.4"
    else
        osname="all"
        echo "Going to terminate $osname"
    fi
fi

if [[ $action == "deploy" ]]; then
    if [[ -z $count ]]; then
        # Default count value to 4 ec2 instances
        count=4
        echo "Default count to $count"
    fi

    if [[ -z $volume_size ]]; then
        # Default volume_size to 30G for RKE2 setup
        volume_size=30
        echo "Default volume_size to $volume_size"
    fi
    if [[ -z $prefix ]]; then
        if [[ -z $PREFIX ]]; then
            echo "Either use -p option to set prefix or set env var/ export PREFIX to skip this option. Cannot proceed with deploy action without this. Exiting the script."
            exit 1
        fi
        echo "Default prefix to the environment variable PREFIX: $PREFIX"
        prefix=$PREFIX
    fi
fi


if [[ -z $key_name ]]; then
    if [[ -z $KEY_NAME ]]; then
        echo "Either set -k value for key_name or set env var/export variable KEY_NAME to skip this option. cannot proceed without this value. Exiting the script."
        exit 1
    fi
    echo "Default key_name to the environement variable KEY_NAME: $KEY_NAME"
    key_name=$KEY_NAME
fi

if [[ $action == "deploy" || $action == "get_running" ]]; then
    if [[ -z $pem_file_path ]]; then
        if [[ -z $PEM_FILE_PATH ]]; then
            echo "Either set -f value for pem_file_path or set env var/export PEM_FILE_PATH to skip this option. Cannot proceed without this value. Exiting the script."
            exit 1
        fi
        echo "Default pem_file_path to the environment variable PEM_FILE_PATH: $PEM_FILE_PATH"
        pem_file_path=$PEM_FILE_PATH
    fi
fi

case $action in
    deploy)
        echo "Deploying OS: $osname ImageID: $image_id SSH_User: $ssh_user" 
        aws ec2 run-instances --image-id $image_id --instance-type $instance_type --count $count --key-name $key_name --security-group-ids sg-0e753fd5550206e55 --block-device-mappings "[{\"DeviceName\":\"/dev/sda1\",\"Ebs\":{\"VolumeSize\":$volume_size,\"DeleteOnTermination\":true}}]" --tag-specifications "[{\"ResourceType\": \"instance\", \"Tags\": [{\"Key\": \"Name\", \"Value\": \"$prefix-$osname\"}]}]" > /dev/null
        sleep 30  # To ensure the system is actually running by the time we use the ssh command output by this script.
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