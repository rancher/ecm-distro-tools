# ECM Distro Tools

ECM Distro Tools (codename: dirt-weasel) is a collection of programs, scripts, and utilities that provide for easier administration, management, and interaction with the RKE2 and K3s ecosystems.

## Building
There's a mix of Go and shell scripts in this repository. The shell scripts reside in the `bin` directory and are ready to use. The Go programs are rooted in the `cmd` directory and need to be compiled. To compile the Go programs run the following in the root of the project:

```sh
make all
```
To compile the container image locally, perform:

```sh
docker build . -t rancher/ecm-distro-tools:local
```
## Utility Index 
The following is a non-exausitve list of the utilities included in this repository and their corresponding usage.
(see bin/ and cmd/ for all utilities.)

### Bump the GO_VERSION in rancher projects
```sh
docker run --rm -it --env GITHUB_TOKEN=<TOKEN> rancher/ecm-distro-tools update_go -o 1.16.3b7 -n 1.17.3b7 -r image-build--envtcd
```
### Create a backport
```sh
docker run --rm -it --env GITHUB_TOKEN=<TOKEN> rancher/ecm-distro-tools backport -r k3s -m v1.21.5+k3s1 -p v1.21.4+k3s1 
```
### Generate release notes
```sh
docker run --rm -it --env GITHUB_TOKEN=<TOKEN> rancher/ecm-distro-tools gen-release-notes -r k3s -m v1.21.5+k3s1 -p v1.21.4+k3s1 
```
### Check for a single kubernetes release
```sh
docker run --rm -it --env GITHUB_TOKEN=<TOKEN> rancher/ecm-distro-tools check_for_k8s_release -r v1.23.3
```

### Check for multiple kubernetes releases
```sh
docker run --rm -it --env GITHUB_TOKEN=<TOKEN> rancher/ecm-distro-tools check_for_k8s_release -r 'v1.23.3 v1.22.6 v1.21.9 v1.20.15'
```

### Create a weekly report for k3s
```sh
docker run --rm -it --env GITHUB_TOKEN=<TOKEN> rancher/ecm-distro-tools -r k3s
```
### Create a weekly report for RKE2
```sh
docker run --rm -it --env GITHUB_TOKEN=<TOKEN> rancher/ecm-distro-tools -r rke2
```

## Contributing

## License

ecm-distro-tools source code is available under the Apache Clause [License](/LICENSE).
