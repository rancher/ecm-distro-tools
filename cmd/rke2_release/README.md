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

```
export GITHUB_TOKEN={YOUR_GITHUB_TOKEN}

rke2_release image-build-base-release
```

## Contributions

- File Issue with details of the problem, feature request, etc.
- Submit a pull request and include details of what problem or feature the code is solving or implementing.
