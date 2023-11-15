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

Optional flags can be automatically set if you are inside your rancher fork. ⚠️ If you decide to run this way, please double check your branch and directory ⚠️

| **Flag**              | **Description**                                                                                                                                                                               | **Required** |
| --------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------ |
| `fork-path`, `f`      | Path for your fork of rancher/rancher. Default is the currenty directory you are running the command in.                                                                                      | FALSE        |
| `base-branch`, `b`    | The branch you want to update with the new KDM branch. Default is the current branch in your repo.                                                                                            | FALSE        |
| `new-kdm-branch`, `n` | KDM branch to replace the current.                                                                                                                                                            | TRUE         |
| `create-pr`, `p`      | if true, will try to create a PR against the `base-branch` in rancher/rancher, may fail if your GitHub token doesn’t have the required permission. Requires a GITHUB_TOKEN env var to be set. | FALSE        |
| `github-user`, `u`    | GitHub Username of the owner of the rancher fork used in `rancher-fork`. Default is the username of the `origin` remote                                                                       | FALSE        |
| `dry-run`, `r`        | Changes will not be pushed to remote and the PR will not be created.                                                                                                                          | FALSE        |

**Examples**

```
rancher_release set-kdm-branch-refs -n dev-v2.8-september-patches --create-pr --dry-run
```

```
export GITHUB_TOKEN={YOUR_GITHUB_TOKEN}

rancher_release set-kdm-branch-refs -n dev-v2.8-september-patches -p -r
```

```
rancher_release set-kdm-branch-refs --fork-path $GOPATH/src/github.com/{YOUR_USERNAME}/rancher \
    --base-branch release/v2.8 \
    --current-kdm-branch dev-v2.8 \
    --new-kdm-branch dev-v2.8-september-patches
```

```
export GITHUB_TOKEN={YOUR_GITHUB_TOKEN}

rancher_release set-kdm-branch-refs -f $GOPATH/src/github.com/{YOUR_USERNAME}/rancher -b release/v2.8 -c dev-v2.8 -n dev-v2.8-september-patches -p -u {YOUR_USERNAME}
```

### set-charts-branch-refs

Updates Rancher branch references in charts:

- `pkg/settings/setting.go`
- `package/Dockerfile`
- `scripts/package-env`

Non-required flags can be automatically set, if you are inside your rancher fork. ⚠️ If you decide to run this way, please double check your branch and directory ⚠️

| **Flag**                 | **Description**                                                                                                                                                                               | **Required** |
| ------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------ |
| `fork-path`, `f`         | Path for your fork of rancher/rancher. Default is the currenty directory you are running the command in.                                                                                      | FALSE        |
| `base-branch`, `b`       | The branch you want to update with the new charts branch. Default is the current branch in your repo.                                                                                         | FALSE        |
| `new-charts-branch`, `n` | New branch to replace the current.                                                                                                                                                            | TRUE         |
| `create-pr`, `p`         | if true, will try to create a PR against the `base-branch` in rancher/rancher, may fail if your GitHub token doesn’t have the required permission. Requires a GITHUB_TOKEN env var to be set. | FALSE        |
| `github-user`, `u`       | GitHub Username of the owner of the rancher fork used in `rancher-fork`. Default is the username of the `origin` remote                                                                       | FALSE        |
| `dry-run`, `r`           | Changes will not be pushed to remote and the PR will not be created.                                                                                                                          | FALSE        |

**Examples**

```
export GITHUB_TOKEN={YOUR_GITHUB_TOKEN}

rancher_release set-charts-branch-refs --new-charts-branch dev-v2.9 --create-pr --dry-run
```

```
export GITHUB_TOKEN={YOUR_GITHUB_TOKEN}

rancher_release set-charts-branch-refs -n dev-v2.9 -p -r
```

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

### check-rancher-rc-deps

This command checks Rancher by the commit hash in the selected files, verifying if they contain 'rc' and 'dev' dependencies. It generates an MD-formatted file print that can be used as a release description. If necessary, the command can generate an error if these dependencies are found, ideal for use in CI pipelines. When executed in the root of the Rancher project, it checks the `rancher-images.txt` and `rancher-windows-images.txt` files.

The pattern of files to be checked includes:
- `pkg/settings/setting.go`
- `package/Dockerfile`
- `scripts/package-env`
- `Dockerfile.dapper`
- `go.mod`
- `pkg/apis/go.mod`
- `pkg/client/go.mod`

| **Flag**         | **Description**                                                                                       | **Required** |
| ---------------- | ----------------------------------------------------------------------------------------------------- | ------------ |
| `commit`, `c`    | Required commit to find the Rancher project reference that will be executed                            | TRUE         |
| `org`, `o`       | Reference organization of the commit                                                                   | FALSE        |
| `repo`, `r`      | Reference repository of the commit                                                                     | FALSE        |
| `files`, `f`     | List of files to be checked by the command, they are mandatory                                         | TRUE         |
| `for-ci`, `p`    | With this flag, it's possible to return an error if any of the files contain 'rc' tags or 'dev' dependencies, ideal for use in integration pipelines | FALSE        |

**Examples**

```
rancher_release check-rancher-rc-deps -c <HASH_COMMIT> -f Dockerfile.dapper,go.mod,/package/Dockerfile,/pkg/apis/go.mod,/pkg/settings/setting.go,/scripts/package-env
```

```
# Images with -rc

* rancher/backup-restore-operator v4.0.0-rc1 (./bin/rancher-images.txt, line 1)
* rancher/rancher v2.8.0-rc3 (./bin/rancher-windows-images.txt, line 1)
* rancher/rancher-agent v2.8.0-rc3 (./bin/rancher-windows-images.txt, line 2)
* rancher/system-agent v0.3.4-rc1-suc (./bin/rancher-windows-images.txt, line 3)

# Components with -rc

* ENV CATTLE_KDM_BRANCH=dev-v2.8 (Dockerfile.dapper, line 16)
* ARG SYSTEM_CHART_DEFAULT_BRANCH=dev-v2.8 (/package/Dockerfile, line 1)
* ARG CHART_DEFAULT_BRANCH=dev-v2.8 (/package/Dockerfile, line 1)
* ARG CATTLE_KDM_BRANCH=dev-v2.8 (/package/Dockerfile, line 1)
* KDMBranch = NewSetting("kdm-branch", "dev-v2.8") (/pkg/settings/setting.go, line 84)
* ChartDefaultBranch = NewSetting("chart-default-branch", "dev-v2.8") (/pkg/settings/setting.go, line 116)
* SYSTEM_CHART_DEFAULT_BRANCH=${SYSTEM_CHART_DEFAULT_BRANCH:-"dev-v2.8"} (/scripts/package-env, line 5)
* CHART_DEFAULT_BRANCH=${CHART_DEFAULT_BRANCH:-"dev-v2.8"} (/scripts/package-env, line 7)

# Min version components with -rc

* ENV CATTLE_FLEET_MIN_VERSION=103.1.0+up0.9.0-rc.3
* ENV CATTLE_CSP_ADAPTER_MIN_VERSION=103.0.0+up3.0.0-rc1

# Components with dev-

* ENV CATTLE_KDM_BRANCH=dev-v2.8 (Dockerfile.dapper, line 16)
* ARG SYSTEM_CHART_DEFAULT_BRANCH=dev-v2.8 (/package/Dockerfile, line 1)
* ARG CHART_DEFAULT_BRANCH=dev-v2.8 (/package/Dockerfile, line 1)
* ARG CATTLE_KDM_BRANCH=dev-v2.8 (/package/Dockerfile, line 1)
* KDMBranch = NewSetting("kdm-branch", "dev-v2.8") (/pkg/settings/setting.go, line 84)
* ChartDefaultBranch = NewSetting("chart-default-branch", "dev-v2.8") (/pkg/settings/setting.go, line 116)
* SYSTEM_CHART_DEFAULT_BRANCH=${SYSTEM_CHART_DEFAULT_BRANCH:-"dev-v2.8"} (/scripts/package-env, line 5)
* CHART_DEFAULT_BRANCH=${CHART_DEFAULT_BRANCH:-"dev-v2.8"} (/scripts/package-env, line 7)

```

## Contributions

- File Issue with details of the problem, feature request, etc.
- Submit a pull request and include details of what problem or feature the code is solving or implementing.
