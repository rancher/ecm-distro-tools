# rke2-charts version references update

Update rke2-charts version references with a single command.

## Examples

```sh
# -c flag is not needed when CHARTS environment variable is present
$ export CHARTS=$(mktemp -d '/tmp/XXXXXX')
$ git clone -o upstream --depth 1 https://github.com/rancher/rke2-charts.git $CHARTS

## print versions
# chart-up <rke2 package name> [no args]
$ chart-up rke2-canal-1.19-1.20
appVersion: v3.13.3
version: v3.13.3-build20211022
rancher/hardened-calico: v3.13.3-build20210223
rancher/hardened-flannel: v0.14.1-build20211022
packageVersion: 5

## print updated version, no file change
# chart-up <rke2 package name> [field=version]
$ chart-up rke2-canal-1.19-1.20 appVersion=v3.13.4
appVersion: v3.13.4
version: v3.13.3-build20211022
rancher/hardened-calico: v3.13.3-build20210223
rancher/hardened-flannel: v0.14.1-build20211022
packageVersion: 5
# Update multiple values including docker image version
$ chart-up rke2-canal-1.19-1.20 rancher/hardened-calico=v3.13.5-build20220124 appVersion=v3.13.4
appVersion: v3.13.4
version: v3.13.3-build20211022
rancher/hardened-calico: v3.13.5-build20220124
rancher/hardened-flannel: v0.14.1-build20211022
packageVersion: 5

## print resulting yaml file into STDOUT
# chart-up -p <rke2 package name> [field=version]
$ chart-up -p rke2-canal-1.19-1.20 appVersion=v3.13.4
apiVersion: v1
appVersion: v3.13.4
[... skipped for brevity ...]

## Write changes into their respective files
# chart-up -i <rke2 package name> [field=version]
$ chart-up -i rke2-canal-1.19-1.20 appVersion=v3.13.4
```
