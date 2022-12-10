# Packer Templates

The json files come built with a source_ami and ami name built in as per packer's requirements, but these can be overwritten via variables passed at runtime. See below for how to run it with passing in those variables.

`packer build -var access_key=YOURKEY -var secret_key=YOURSECRET -var source_ami=SOURCE_AMI_ID -var name=DESIRED_NAME file.json"`
