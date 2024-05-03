# ECM Distro Tools

ECM Distro Tools is a collection of utilities that provide for easier administration, management, and interaction with the great Rancher ecosystems, including RKE2 and K3s.

## Building

There's a mix of code in this repository. The shell scripts and shell libraries reside in the `bin` directory and are ready to use. The Go programs are rooted in the `cmd` directory and need to be compiled.

To compile the programs run the following in the root of the project:

```sh
make all
```

To compile the container image locally:

```sh
docker build . -t rancher/ecm-distro-tools
```

## Installation

The easiest way to install a single utility is to go to the release page, choose the release you want, and download the utility for your operation system and architecture.

### Install Script

To install all executables and shell libraries, run the install script as follows:

```sh
# to install the latest version
curl -sfL https://raw.githubusercontent.com/rancher/ecm-distro-tools/master/install.sh | sh -
# or
./install.sh
# to install a specific version
curl -sfL https://raw.githubusercontent.com/rancher/ecm-distro-tools/master/install.sh | ECM_VERSION=v0.31.2 sh -
# or
./install.sh v0.31.2
```

This will download all binaries and shell libraries and install them to `/usr/local/bin/ecm-distro-tools`. You'll need to add that directory to your path after installation.

## Utility Index

The following is a non-exausitve list of the utilities included in this repository and their corresponding usage.
(see bin/ and cmd/ for all utility code)

### List available utilities

```sh
utility_index
```

For details on specific utilities, review the script header or the README for the specific utility.

All utilities comes with help output.

## GitHub action

This repository provides the "Setup ecm-distro-tools" GitHub action.
It downloads the assets belonging to the specified release to a temporary directory,
and adds the directory to the `PATH`.

### Usage

The action can be run on ubuntu-latest runners.
The `version` parameter is required.
Providing the GH_TOKEN environment variable is recommended to avoid rate limiting by the GitHub API.

```yaml
steps:
  - name: setup ecm-distro-tools
    uses: rancher/ecm-distro-tools@v0.27.0
    with:
      version: v0.27.0
    env:
      GH_TOKEN: ${{ github.token }}
  - name: release
    run: release -h
```

## Contributing

We welcome additions to this repo and are excited to keep expanding its functionality.

To contribute, please do the following:

### Features and Bugs

- Open an issue explaining the feature(s) / bug(s) you are looking to add/fix.
- Fork the repo, create a branch, push your changes, and open a pull request.
- Request review

## Development

### Expectations

A set of patterns have been established with the Go and shell code that need to be adhered to. Usage output and flags should be copied and pasted from other code files and adjusted to keep the UX as similar as possible to the rest of the utilities in the repo.

Library code has been written for Go and shell which to simpler access to Github, loggers, and means of validating common checks.

When a new utility is added or an API is changed, documentation needs to be updated to reflect that change. This needs to be done wherever that documentation lives, likely the utility's README.

#### Go

- Go code additions are expected to have been linted, vetted, and fmt'd prior to pushing the code.
- Prefer the standard library over 3rd party libraries when possible

#### Shell

- Shell scripts are expected to be POSIX compliant, avoiding specific shell features for portability. We are currently using `shellcheck` to perform these checks and validations.

#### Building

When building locally, you may want to build just for your ARCH and OS. To do so, you can use one of the two methods below:

Using this method, you can set these variables in `.bashrc` or `.zshrc` to make it easier to alaways build for your ARCH and OS.
Just be aware that if you do this, you'll need to unset them if you want to build for all ARCHs and OSs.

```sh
export ARCHS=amd64
export OSs=linux
make all
```

Using this method, you'll need to set the variables each time you want to build just for your ARCH and OS.

```sh
make ARCHS=amd64 OSs=linux all
```

## License

ecm-distro-tools source code is available under the Apache Clause [License](/LICENSE).
