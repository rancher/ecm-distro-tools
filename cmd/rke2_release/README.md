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

## Contributions

- File Issue with details of the problem, feature request, etc.
- Submit a pull request and include details of what problem or feature the code is solving or implementing.
