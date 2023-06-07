# ECM Distro Tools

ECM Distro Tools (codename: dirt-weasel) is a collection of programs, scripts, and utilities that provide for easier administration, management, and interaction with the RKE2 and K3s ecosystems.

## Building

There's a mix of C, Go and shell scripts in this repository. The shell scripts reside in the `bin` directory and are ready to use. The Go programs are rooted in the `cmd` directory and need to be compiled. To compile the Go programs run the following in the root of the project:

```sh
make all
```

To compile the container image locally, perform:

```sh
docker build . -t rancher/ecm-distro-tools
```

## Utility Index

The following is a non-exausitve list of the utilities included in this repository and their corresponding usage.
(see bin/ and cmd/ for all utility code)

### List available utilities

```sh
docker run --rm -it rancher/ecm-distro-tools utility_index
```

### Bump the GO_VERSION in rancher projects

```sh
docker run --rm -it --env GITHUB_TOKEN=<TOKEN> rancher/ecm-distro-tools update_go -o 1.16.3b7 -n 1.17.3b7 -r image-build--envtcd
```

### Create a backport for k3s or rke2

```sh
docker run --rm -it --env GITHUB_TOKEN=<TOKEN> rancher/ecm-distro-tools backport -r k3s -m v1.21.5+k3s1 -p v1.21.4+k3s1 
```

### Generate release notes for k3s or rke2

```sh
docker run --rm -it --env GITHUB_TOKEN=<TOKEN> rancher/ecm-distro-tools gen-release-notes -r k3s -m v1.21.5+k3s1 -p v1.21.4+k3s1 
```

### Check for kubernetes releases

```sh
docker run --rm -it --env GITHUB_TOKEN=<TOKEN> rancher/ecm-distro-tools check_for_k8s_release -r 'v1.23.3 v1.22.6 v1.21.9 v1.20.15'
```

### Create a weekly report for k3s or rke2

```sh
docker run --rm -it --env GITHUB_TOKEN=<TOKEN> rancher/ecm-distro-tools weekly_report -r k3s
```

### Daily Standup Template Generator

Send template output to standard out.

```sh
docker run --rm -it rancher/ecm-distro-tools standup
```

Send template output to a file at `${PWD}`. File will be named `YYYY-MM-dd`

```sh
docker run --rm -it rancher/ecm-distro-tools standup -f
```

### Retrieve Bootstrap Hash

```sh
docker run --rm -it rancher/ecm-distro-tools bootstrap_hash -p k3s
```

### Scan an image the same as Rancher

```sh
docker run --rm -it --env GITHUB_TOKEN=<TOKEN> rancher/ecm-distro-tools rancher_image_scan <IMAGE_NAME>
```

### Create tag for RKE2 component release

```sh
# RKE2 Kubernetes base image
docker run --rm -it --env GITHUB_TOKEN=<TOKEN> rancher/ecm-distro-tools tag_image_build_k8s_release -d -r 'v1.23.2-rke2r1'
```

```sh
# RKE2
docker run --rm -it --env GITHUB_TOKEN=<TOKEN> rancher/ecm-distro-tools tag_rke2_release -d -m '1.23' -r 'v1.23.5-rc1+rke2r1'
```

```sh
# RKE2 packaging (rpms)
docker run --rm -it --env GITHUB_TOKEN=<TOKEN> rancher/ecm-distro-tools tag_rke2_packaging_release -d -t -r 'v1.22.6+rke2r1'
```

### Test-Pad Tool

See seperate [documentation](./docs/test_pad.md) for test-pad tool.

## Contributing

We welcome additions to this repo and are excited to keep expanding its functionality.

To contribute, please do the following:

### Features and Bugs

* Open an issue explaining the feature(s) / bug(s) you are looking to add/fix.
* Fork the repo, create a branch, push your changes, and open a pull request.
* Request review

### Expectations

A set of patterns have been established with the Go and shell code that will need to be adhered to. Usage output and flags should be copied and pasted from other code files and adjusted to keep the UX as similar as possible to the rest of the utilities in the repo.

Library code has been written for Go and shell which to simpler access to Github, loggers, and means of validating common checks.

#### Go

* Go code additions are expected to have been linted, vetted, and fmt'd prior to pushing the code. 
* Prefer the standard library over 3rd party libraries when possible

#### Shell

* Shell scripts are expected to be POSIX compliant, avoiding specific shell features for portability. We are currently using `shellcheck` to perform these checks and validations.

## License

ecm-distro-tools source code is available under the Apache Clause [License](/LICENSE).
