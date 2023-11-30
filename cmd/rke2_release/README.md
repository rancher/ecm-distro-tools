# rke2_release

rke2_release is a utility that performs the bulk of the actions to complete a rke2 release.

Please reference the help menu from the binary.

## Commands

### image-build-base-release

Checks if new Golang versions are available and creates new releases in `rancher/image-build-base`

| **Flag**                            | **Description** | **Required** |
| ----------------------------------- | --------------- | ------------ |
| `github-token`, `g`, `GITHUB_TOKEN` | Github Token    | TRUE         |

**Examples**

```sh
export GITHUB_TOKEN={YOUR_GITHUB_TOKEN}

rke2_release image-build-base-release
```

```sh
rke2_release components -l all
image-build-base                            |v1.21.3b1
image-build-calico                          |v3.26.1-build20231009
image-build-cni-plugins                     |v1.2.0-build20231009
...
```

```sh
rke2_release components -l image-build-k8s-metrics-server
image-build-k8s-metrics-server|v0.6.3-build20231009
```

### image-build

#### list-repos
List all repos supported by this command

```sh
rke2_release image-build list-repos

> INFO[0000] Supported repos: 
    image-build-sriov-network-resources-injector
    image-build-containerd
    image-build-multus
    image-build-ib-sriov-cni
    image-build-sriov-network-device-plugin
    image-build-cni-plugins
    image-build-rke2-cloud-provider
    image-build-dns-nodecache
    image-build-k8s-metrics-server
    image-build-calico
    image-build-flannel
    image-build-sriov-cni
    image-build-whereabouts
    image-build-etcd
    image-build-runc  
```

#### update
Updates references to the `hardened-build-base` docker image in the `rancher/image-build-*` repos.

| **Flag**                            | **Description** | **Required** |
| ----------------------------------- | --------------- | ------------ |
| `github-token`, `g`, `GITHUB_TOKEN` | Github Token    | TRUE         |
| `repo`, `r` | What image-build repo to update, use `rke2_release image-build list-repos` for a full list    | TRUE         |
| `owner`, `o` |  Owner of the repo, default is 'rancher' only used for testing purposes   | FALSE         |
| `repo-path`, `p` |  Local copy of the image-build repo that is being updated   | TRUE         |
| `working-dir`, `w` |  Directory in which the temporary scripts will be created, default is /tmp   | FALSE         |
| `build-base-tag`, `t` |   hardened-build-base Docker image tag to update the references in the repo to  | TRUE         |
| `dry-run`, `d` |  Don't push changes to remote and don't create the PR, just log the information   | FALSE         |
| `create-pr`, `c` |  If not set, the images will be pushed to a new branch, but a PR won't be created  | FALSE         |

**Examples**

```sh
export GITHUB_TOKEN={YOUR_GITHUB_TOKEN}

rke2_release image-build update --repo image-build-calico --repo-path /tmp/image-build-calico --build-base-tag v1.21.3b1 --create-pr --dry-run
```

```sh
export GITHUB_TOKEN={YOUR_GITHUB_TOKEN}

rke2_release image-build update -r image-build-calico -p /tmp/image-build-calico -t v1.21.3b1 -c -d
```
 
### Components

```sh
rke2_release components -l all
image-build-base                            |v1.21.3b1
image-build-calico                          |v3.26.1-build20231009
image-build-cni-plugins                     |v1.2.0-build20231009
...
```

```sh
rke2_release components -l image-build-k8s-metrics-server
image-build-k8s-metrics-server|v0.6.3-build20231009
```

## Contributions

- File Issue with details of the problem, feature request, etc.
- Submit a pull request and include details of what problem or feature the code is solving or implementing.
