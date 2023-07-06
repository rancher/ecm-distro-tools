AWS EC2 Instance Manager - Deploy/Terminate Helper Script

Pre-Requisites:

    1. Setup awscli in your environment:
        Run "aws configure" command on your mac. Ensure the following values are setup:
    ```
    # aws configure
    AWS Access Key ID [****************MGHV]: 
    AWS Secret Access Key [****************7zvs]: 
    Default region name [us-east-2]: 
    Default output format []:
    ```
    2. Setup Environment variables(if you dont want to provide these options via cmdline args every time you use the script):
    ```
    export PREFIX="prefix"  # Set this to your name abbreviation. Used to tag the ec2 instances to identify they are yours
    export KEY_NAME="ssh-key-pair-name"  # Set the key-pair login information to connect securely to the ec2 instances launched
    export KEY_FILE_PATH="/path/to/your/name.pem"  # Set the full file path to your .pem file. Needed to ssh to your instances

    ```

Usage:

```         
    $(basename "$0") [-l] [-d] [-t] [-g] [-o osname] [-p prefix] [-k key_name] [-f pem_file_path] [-c count] [-v volume_size] [-h]

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
```

Examples:

Assuming PREFIX, KEY_NAME and PEM_FILE_PATH are already exported: 
```
    bash aws.sh -h  -> For Usage/Help information

    bash aws.sh -d -> Deploy ubuntu22.04 by default. 
    bash aws.sh -t -> terminate all instances previously deployed
    bash aws.sh -g -> Get Running instance ssh info for default ubuntu22.4 setups
    
    bash aws.sh -d -o rhel9 -> Deploy OS RHEL Version 9 Architecture: x86, with count 4, volume 30GB
    bash aws.sh -d -o rhel9_arm -c 1 -v 20 -> Deploy rhel 9 arm architecture image - count with 1 instance, volume 20GB for EBS.
```

When PREFIX, KEY_NAME and PEM_FILE_PATH are not already exported:

Deploy Action: 
```
    bash aws.sh -d -o rhel9_arm -c 1 -v 20 -p team-rke2-k3s -k key-pair-name -f /path/to/pem/file.pem
    bash aws.sh -d -o rocky8.7 -p team-rke2-k3s -k key-pair-name -f /path/to/pem/file.pem
```
Terminate Action:
```
    bash aws.sh -t -o sles15sp4 -k key-pair-name
```
Get_Running Action:
```
    bash aws.sh -g -o sles15sp4 -k key-pair-name -f /path/to/pem/file.pem
```




