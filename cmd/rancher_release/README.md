# rancher_release

rancher_release is a utility that performs a number of tasks related to the Rancher release.

Please reference the help menu from the binary.

## Commands

### list-nonmirrored-rc-images

Lists all non mirrored images in RC form in a given rancher release, these are extracted from the images.txt artifact attached to a GitHub release.  
Results are printed in MD, and can be pasted into Slack, but formatting is tricky, you’ll see a pop-up asking if you would like to format the text. If you click never ask me again, you’ll need to go to options, advanced and use MD format.

| **Flag** | **Description**        | **Required** |
| -------- | ---------------------- | ------------ |
| tag      | Release tag in GitHub. | TRUE         |

**Examples**

```
rancher_release list-nonmirrored-rc-images —tag v2.6
```

### check-rancher-image

Checks if there’s an available Helm Chart and Docker images for amd64, arm and s390x for a given tag.

| **Flag** | **Description**        | **Required** |
| -------- | ---------------------- | ------------ |
| tag      | Release tag in GitHub. | TRUE         |

**Examples**

```
rancher_release check-rancher-image —tag v2.6
```

### set-kdm-branch-refs

Updates Rancher KDM branch references in:

- `pkg/settings/setting.go`
- `package/Dockerfile`
- `Dockerfile.dapper`

| **Flag**           | **Description**                                                                                                                                                                               | **Required** |
| ------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------ |
| fork-path          | Path for your fork of rancher/rancher.                                                                                                                                                        | TRUE         |
| base-branch        | The branch you want to update with the new KDM branch.                                                                                                                                        | TRUE         |
| current-kdm-branch | Current KDM branch used in the files listed above.                                                                                                                                            | TRUE         |
| new-kdm-branch     | KDM branch to replace the current.                                                                                                                                                            | TRUE         |
| create-pr          | if true, will try to create a PR against the `base-branch` in rancher/rancher, may fail if your GitHub token doesn’t have the required permission. Requires a GITHUB_TOKEN env var to be set. | FALSE        |
| fork-owner         | GitHub Username of the owner of the rancher fork used in `rancher-fork`.                                                                                                                      | FALSE        |
| dry-run            | Changes will not be pushed to remote and the PR will not be created.                                                                                                                          | FALSE        |

**Examples**

```
rancher_release set-kdm-branch-refs —fork-path $GOPATH/src/github.com/{YOUR_USERNAME}/rancher \
    —base-branch release/v2.8 \
    —current-kdm-branch dev-v2.8 \
    —new-kdm-branch dev-v2.8-september-patches
```

```
export GITHUB_TOKEN={YOUR_GITHUB_TOKEN}

rancher_release set-kdm-branch-refs —fork-path $GOPATH/src/github.com/{YOUR_USERNAME}/rancher \
    —base-branch release/v2.8 \
    —current-kdm-branch dev-v2.8 \
    —new-kdm-branch dev-v2.8-september-patches \
    —create-pr \
    —fork-owner {YOUR_USERNAME}
```

### set-charts-branch-refs

Updates Rancher branch references in charts:

- `pkg/settings/setting.go`
- `package/Dockerfile`
- `Dockerfile.dapper`

| **Flag**              | **Description**                                                                                                                                                                               | **Required** |
| --------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------ |
| fork-path             | Path for your fork of rancher/rancher.                                                                                                                                                        | TRUE         |
| base-branch           | The branch you want to update.                                                                                                                                                                | TRUE         |
| current-charts-branch | Current branch for charts used in the files listed above.                                                                                                                                     | TRUE         |
| new-charts-branch     | Branch to replace the current in the charts.                                                                                                                                                  | TRUE         |
| create-pr             | if true, will try to create a PR against the `base-branch` in rancher/rancher, may fail if your GitHub token doesn’t have the required permission. Requires a GITHUB_TOKEN env var to be set. | FALSE        |
| fork-owner            | GitHub Username of the owner of the rancher fork used in `rancher-fork`.                                                                                                                      | FALSE        |
| dry-run               | Changes will not be pushed to remote and the PR will not be created.                                                                                                                          | FALSE        |

**Examples**

```
rancher_release set-charts-branch-refs —fork-path $GOPATH/src/github.com/{YOUR_USERNAME}/rancher \
    —base-branch release/v2.8 \
    —current-charts-branch dev-v2.8 \
    —new-charts-branch dev-v2.9
```

```

export GITHUB_TOKEN={YOUR_GITHUB_TOKEN}

rancher_release set-charts-branch-refs —fork-path $GOPATH/src/github.com/{YOUR_USERNAME}/rancher \
    —base-branch release/v2.8 \
    —current-charts-branch dev-v2.8 \
    —new-charts-branch dev-v2.9 \
    —create-pr \
    —fork-owner {YOUR_USERNAME}
```

## Contributions

- File Issue with details of the problem, feature request, etc.
- Submit a pull request and include details of what problem or feature the code is solving or implementing.