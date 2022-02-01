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
### Using the docker image

```sh
docker run --rm --it -e GITHUB_TOKEN=<token> rancher/ecm-distro-tools <utility> <args>
```
### update_go

```sh
usage: /usr/local/bin/update_go [onrsph]
    -o    version     
    -n    version
    -r    repository (rancher/image-build-base)
    -s    display the hardened-build-base version of a target repository
    -p    path to clone repositories 
    -h    show help

examples: 
    /usr/local/bin/update_go -o 1.16.3b7 -n 1.17.3b7 -r image-build-etcd
```
```
### backport
```sh
version: v0.1.0
Usage: backport [-t token] [-r repo] [-m milestone] 
Options:
    -h                   help
    -v                   show version and exit
    -t                   github token (optional)
    -r repo              repository that should be used
    -i issue id          original issue id
    -c commit            commit id that is being bacported
    -b branch(es)        branches issue is being backported to
Examples: 
    backport -r k3s -m v1.21.5+k3s1 -p v1.21.4+k3s1 
```
### gen-release-notes
```sh
version: v0.4.0
Usage: gen-release-notes [-r repo] [-m milestone] [-p prev milestone]
Options:
    -h                   help
    -v                   show version and exit
    -t                   github token (optional)
    -r repo              repository that should be used
    -m milestone         milestone to be used
        -p prev milestone    previous milestone
Examples: 
    gen-release-notes -r k3s -m v1.21.5+k3s1 -p v1.21.4+k3s1 
```

## Contributing

## License

ecm-distro-tools source code is available under the Apache Clause [License](/LICENSE).
