#! /bin/sh
# note this was written using aws cli version
# aws-cli/1.27.97 Python/3.10.10 Darwin/22.3.0 botocore/1.29.97

# consider KEY_NAME global variable for ease of use
make_aws() {
	_instance_count="${1:-1}" # number of instances to make each run of the script
	_ami="${2:-0535d9b70179f9734}" # sles 15sp4
	_instance_type="${3:-t2.medium}"
	_ebs_size="${4:-35}"
	_key_name="${5:-$AWS_KEY_NAME}" #sometimes region dependent
	_random_num=$(jot -r 1 10 99)
	_name_tag='ResourceType=instance,Tags=[{Key=Name,Value=YOUR-NAME-'$_random_num'}]'
	#local path_to_config="file://~/path/to/configs/cloud-config.yaml"
	aws ec2 run-instances \
	--image-id ami-"$_ami" \
	--count "$_instance_count" \
	--instance-type "$_instance_type" \
	--key-name "$_key_name" \
	--security-group-ids sg-security-group-id \
	--subnet-id subnet-subnet-id \
	--block-device-mapping DeviceName=/dev/sda1,Ebs={VolumeSize="$_ebs_size"} \
	--tag-specifications "$_name_tag" \
	#--user-data "$path_to_config"
}

get_aws() {
    _greppies="${1:-pub}"
	_keyName="${2:-$AWS_KEY_NAME}"
	_region="${3:-us-east-2}"
    case "$_greppies" in
    all) aws ec2 --region "$_region" describe-instances --filter Name=key-name,Values="$_keyName" | grep -e InstanceId -e PublicIpAddress -e PrivateIpAddress -e Ipv6Address ;;
    pub) aws ec2 --region "$_region" describe-instances --filter Name=key-name,Values="$_keyName" | grep -e PublicIpAddress | awk '{print $2 ":    " $4}' ;;
    esac
}

drop_aws() { 
	aws ec2 terminate-instances \
	--instance-ids "$1" "$2" "$3" "$4" "$5"
}

drop_AAWS() {
	_keyName="${1:-$AWS_KEY_NAME}"
	# shellcheck disable=SC2000-SC4000
	if [ -z "$_keyName" ]
	then
		echo "keyName variable is empty..."
	else
	    _ARRAY=($(aws ec2 describe-instances --filter Name=key-name,Values="$_keyName" | grep -i -e InstanceId | awk '{print $4}'));
		echo "$_ARRAY"
    	for id in "${_ARRAY[@]}"
			do
				drop_aws "$id" 
			done
	fi
}

get_mylb() {
	_prefixer="${1:-usually-firstname-twice}"
	aws elbv2 describe-load-balancers | grep -e "$_prefixer" | grep -e LoadBalancerArn | awk '{print $2};END{print NR " Total LoadBalancers"}'
}

get_awslb() {
	aws elbv2 describe-load-balancers | grep -e LoadBalancerArn | awk '{print $2 ":    " $4};END{print NR " Total LoadBalancers"}' | sort
}

get_lbname(){
	 aws elbv2 describe-load-balancers | grep -i -e LoadBalancerName | awk '{print $2 ":  " $4};END{print NR " Total LoadBalancers"}' | sort
}

drop_awslb(){
	aws elbv2 delete-load-balancer --load-balancer-arn "$1" # aws cli doesn't seem to support multiple arns? 
}
