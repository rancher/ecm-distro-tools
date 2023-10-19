# rancher_release

rancher_release is a utility that performs a number of tasks related to the Rancher release.

Please reference the help menu from the binary.

## Commands

### list-nonmirrored-rc-images

Lists all non mirrored images in RC form in a given rancher release, these are extracted from the images.txt artifact attached to a GitHub release.  
Results are printed in MD, and can be pasted into Slack, but formatting is tricky, you’ll see a pop-up asking if you would like to format the text. If you click never ask me again, you’ll need to go to options, advanced and use MD format.

| **Flag**   | **Description**        | **Required** |
| ---------- | ---------------------- | ------------ |
| `tag`, `t` | Release tag in GitHub. | TRUE         |

**Examples**

```
rancher_release list-nonmirrored-rc-images --tag v2.8.0-rc1
```

### check-rancher-image

Checks if there’s an available Helm Chart and Docker images for amd64, arm and s390x for a given tag.

| **Flag**   | **Description**        | **Required** |
| ---------- | ---------------------- | ------------ |
| `tag`, `t` | Release tag in GitHub. | TRUE         |

**Examples**

```
rancher_release check-rancher-image --tag v2.8.0-rc1
```

### set-kdm-branch-refs

Updates Rancher KDM branch references in:

- `pkg/settings/setting.go`
- `package/Dockerfile`
- `Dockerfile.dapper`

| **Flag**                  | **Description**                                                                                                                                                                               | **Required** |
| ------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------ |
| `fork-path`, `f`          | Path for your fork of rancher/rancher.                                                                                                                                                        | TRUE         |
| `base-branch`, `b`        | The branch you want to update with the new KDM branch.                                                                                                                                        | TRUE         |
| `current-kdm-branch`, `c` | Current KDM branch used in the files listed above.                                                                                                                                            | TRUE         |
| `new-kdm-branch`, `n`     | KDM branch to replace the current.                                                                                                                                                            | TRUE         |
| `create-pr`, `p`          | if true, will try to create a PR against the `base-branch` in rancher/rancher, may fail if your GitHub token doesn’t have the required permission. Requires a GITHUB_TOKEN env var to be set. | FALSE        |
| `fork-owner`, `o`         | GitHub Username of the owner of the rancher fork used in `rancher-fork`.                                                                                                                      | FALSE        |
| `dry-run`, `r`            | Changes will not be pushed to remote and the PR will not be created.                                                                                                                          | FALSE        |

**Examples**

```
rancher_release set-kdm-branch-refs --fork-path $GOPATH/src/github.com/{YOUR_USERNAME}/rancher \
    --base-branch release/v2.8 \
    --current-kdm-branch dev-v2.8 \
    --new-kdm-branch dev-v2.8-september-patches
```

```
export GITHUB_TOKEN={YOUR_GITHUB_TOKEN}

rancher_release set-kdm-branch-refs -f $GOPATH/src/github.com/{YOUR_USERNAME}/rancher -b release/v2.8 -c dev-v2.8 -n dev-v2.8-september-patches -p -o {YOUR_USERNAME}
```

### set-charts-branch-refs

Updates Rancher branch references in charts:

- `pkg/settings/setting.go`
- `package/Dockerfile`
- `Dockerfile.dapper`

| **Flag**                     | **Description**                                                                                                                                                                               | **Required** |
| ---------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------ |
| `fork-path`, `f`             | Path for your fork of rancher/rancher.                                                                                                                                                        | TRUE         |
| `base-branch`, `b`           | The branch you want to update.                                                                                                                                                                | TRUE         |
| `current-charts-branch`, `c` | Current branch for charts used in the files listed above.                                                                                                                                     | TRUE         |
| `new-charts-branch`, `n`     | Branch to replace the current in the charts.                                                                                                                                                  | TRUE         |
| `create-pr`, `p`             | if true, will try to create a PR against the `base-branch` in rancher/rancher, may fail if your GitHub token doesn’t have the required permission. Requires a GITHUB_TOKEN env var to be set. | FALSE        |
| `fork-owner`, `o`            | GitHub Username of the owner of the rancher fork used in rancher-fork.                                                                                                                        | FALSE        |
| `dry-run`, `r`               | Changes will not be pushed to remote and the PR will not be created.                                                                                                                          | FALSE        |

**Examples**

```
rancher_release set-charts-branch-refs --fork-path $GOPATH/src/github.com/{YOUR_USERNAME}/rancher \
    --base-branch release/v2.8 \
    --current-charts-branch dev-v2.8 \
    --new-charts-branch dev-v2.9
```

```

export GITHUB_TOKEN={YOUR_GITHUB_TOKEN}

rancher_release set-charts-branch-refs -f $GOPATH/src/github.com/{YOUR_USERNAME}/rancher -b release/v2.8 -c dev-v2.8 -n dev-v2.9 -p -o {YOUR_USERNAME}
```

### label-issues

Given a release candidate, updates each GitHub issue belonging to its milestone with the tag `[zube]: To Test` and adds a comment with the prerelease version to test.

**Examples**

```sh
rancher_release label-issues -t v2.8.1-rc1 --dry-run
# Updating 2 issues
# #1 Issue one (v2.8.x)
#   [Waiting for RC] -> [To Test] 
# #2 Issue two (v2.8.x)
#   [Waiting for RC] -> [To Test] 
```


## Contributions

- File Issue with details of the problem, feature request, etc.
- Submit a pull request and include details of what problem or feature the code is solving or implementing.
