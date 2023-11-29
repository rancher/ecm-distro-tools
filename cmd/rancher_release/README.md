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

```sh
rancher_release list-nonmirrored-rc-images --tag v2.8.0-rc1
```

### check-rancher-image

Checks if there’s an available Helm Chart and Docker images for amd64, arm and s390x for a given tag.

| **Flag**   | **Description**        | **Required** |
| ---------- | ---------------------- | ------------ |
| `tag`, `t` | Release tag in GitHub. | TRUE         |

**Examples**

```sh
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

```sh
rancher_release set-kdm-branch-refs -n dev-v2.8-september-patches --create-pr --dry-run
```

```sh
export GITHUB_TOKEN={YOUR_GITHUB_TOKEN}

rancher_release set-kdm-branch-refs -n dev-v2.8-september-patches -p -r
```

```sh
rancher_release set-kdm-branch-refs --fork-path $GOPATH/src/github.com/{YOUR_USERNAME}/rancher \
    --base-branch release/v2.8 \
    --current-kdm-branch dev-v2.8 \
    --new-kdm-branch dev-v2.8-september-patches
```

```sh
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

```sh
export GITHUB_TOKEN={YOUR_GITHUB_TOKEN}

rancher_release set-charts-branch-refs --new-charts-branch dev-v2.9 --create-pr --dry-run
```

```sh
export GITHUB_TOKEN={YOUR_GITHUB_TOKEN}

rancher_release set-charts-branch-refs -n dev-v2.9 -p -r
```

```sh
rancher_release set-charts-branch-refs --fork-path $GOPATH/src/github.com/{YOUR_USERNAME}/rancher \
 --base-branch release/v2.8 \
 --current-charts-branch dev-v2.8 \
 --new-charts-branch dev-v2.9

```

```sh
export GITHUB_TOKEN={YOUR_GITHUB_TOKEN}

rancher_release set-charts-branch-refs -f $GOPATH/src/github.com/{YOUR_USERNAME}/rancher -b release/v2.8 -c dev-v2.8 -n dev-v2.9 -p -o {YOUR_USERNAME}

```

### tag-release
Tags releases in GitHub for Rancher.

When tagging a new release using the `tag-release` command, always prefer to use the default behavior of creating as a draft and verifying the release in the UI before publishing it.
If you are running this locally, you'll need to generate a GitHub Token, use the fine-grained personal access token, scoped to only the rancher repo and with the `contents read and write` scope.


| **Flag**                 | **Description**                                                                                                                                                                               | **Required** |
| ------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------ |
| `github-token`, `g`, `GITHUB_TOKEN`         | GitHub generated token as described above.                                                                                      | TRUE        |
| `tag`, `t`       | The tag that you want to create.                                                                                         | TRUE        |
| `remote-branch`, `b` | The branch which you want to create the tag against.                                                                                                                                                            | TRUE         |
| `repo-owner`, `o`         | Username of the rancher repo owner. Default is `rancher`, only customize this for testing purposes. | FALSE        |
| `general-availability`, `a`         | By default, the release will be created as a pre-release, before setting this as true, make sure it absolutely needs to be a GA release. | FALSE        |
| `ignore-draft`, `d`         | By default, the release will be created as a draft, so you can verify everything is correct before publishing it. | FALSE        |
| `dry-run`, `r`           | The release will not be created, just logged.                                                                                                                          | FALSE        |

**Examples**

```sh
export GITHUB_TOKEN={YOUR_GITHUB_TOKEN}

rancher_release tag-release --tag v2.8.0-rc1 --remote-branch release/v2.8 --dry-run
```

```sh
export GITHUB_TOKEN={YOUR_GITHUB_TOKEN}

rancher_release tag-release --tag v2.8.0-rc1 --remote-branch release/v2.8 --repo-owner tashima42 --dry-run
```

```sh
export GITHUB_TOKEN={YOUR_GITHUB_TOKEN}

rancher_release tag-release -t v2.8.0 -b release/v2.8 -a -r
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

This command checks Rancher verifying if contains 'rc' and 'dev' dependencies for some files, this command could be used locally or remotely by commit hash. It generates an MD-formatted file print that can be used as a release description. If necessary, the command can generate an error if these dependencies are found, ideal for use in CI pipelines. 

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
| `commit`, `c`    | Commit used to get all files during the check, required for remote execution                           | FALSE         |
| `org`, `o`       | Reference organization of the commit, as default `rancher`                                                                                                                            | FALSE        |
| `repo`, `r`      | Reference repository of the commit, as default `rancher`                                                                     | FALSE        |
| `files`, `f`     | List of files to be checked by the command                                   | FALSE         |
| `for-ci`, `p`    | With this flag, it's possible to return an error if any of the files contain 'rc' tags or 'dev' dependencies, ideal for use in integration pipelines | FALSE        |

**Examples**
LOCAL
```
rancher_release check-rancher-rc-deps
```
REMOTE
```
rancher_release check-rancher-rc-deps -c <HASH_COMMIT> -f Dockerfile.dapper,go.mod,/package/Dockerfile,/pkg/apis/go.mod,/pkg/settings/setting.go,/scripts/package-env
```

```
# Images with -rc

* rancher/backup-restore-operator v4.0.0-rc1 (/bin/rancher-images.txt, line 1)
* rancher/fleet v0.9.0-rc.5 (/bin/rancher-images.txt, line 1)
* rancher/fleet-agent v0.9.0-rc.5 (/bin/rancher-images.txt, line 1)
* rancher/rancher v2.8.0-rc3 (/bin/rancher-windows-images.txt, line 1)
* rancher/rancher-agent v2.8.0-rc3 (/bin/rancher-windows-images.txt, line 1)
* rancher/system-agent v0.3.4-rc1-suc (/bin/rancher-windows-images.txt, line 1)

# Components with -rc

* github.com/opencontainers/image-spec => github.com/opencontainers/image-spec v1.1.0-rc2 // needed for containers/image/v5 (go.mod, line 15)
* github.com/rancher/aks-operator v1.2.0-rc4 (go.mod, line 111)
* github.com/rancher/dynamiclistener v0.3.6-rc3-deadlock-fix-revert (go.mod, line 114)
* github.com/rancher/eks-operator v1.3.0-rc3 (go.mod, line 115)
* github.com/rancher/gke-operator v1.2.0-rc2 (go.mod, line 117)
* github.com/rancher/rke v1.5.0-rc5 (go.mod, line 124)
* github.com/opencontainers/image-spec v1.1.0-rc3 // indirect (go.mod, line 370)
* ENV CATTLE_RANCHER_WEBHOOK_VERSION=103.0.0+up0.4.0-rc9 (/package/Dockerfile, line 26)
* ENV CATTLE_CSP_ADAPTER_MIN_VERSION=103.0.0+up3.0.0-rc1 (/package/Dockerfile, line 27)
* ENV CATTLE_CLI_VERSION v2.8.0-rc1 (/package/Dockerfile, line 48)
* github.com/rancher/aks-operator v1.2.0-rc4 (/pkg/apis/go.mod, line 11)
* github.com/rancher/eks-operator v1.3.0-rc3 (/pkg/apis/go.mod, line 12)
* github.com/rancher/gke-operator v1.2.0-rc2 (/pkg/apis/go.mod, line 14)
* github.com/rancher/rke v1.5.0-rc5 (/pkg/apis/go.mod, line 16)
* ShellImage = NewSetting("shell-image", "rancher/shell:v0.1.21-rc1") (/pkg/settings/setting.go, line 121)

# Min version components with -rc

* ENV CATTLE_FLEET_MIN_VERSION=103.1.0+up0.9.0-rc.3
* ENV CATTLE_CSP_ADAPTER_MIN_VERSION=103.0.0+up3.0.0-rc1

# KDM References with dev branch

* ENV CATTLE_KDM_BRANCH=dev-v2.8 (Dockerfile.dapper, line 16)
* ARG CATTLE_KDM_BRANCH=dev-v2.8 (/package/Dockerfile, line 1)
* KDMBranch = NewSetting("kdm-branch", "dev-v2.8") (/pkg/settings/setting.go, line 84)

# Chart References with dev branch

* ARG SYSTEM_CHART_DEFAULT_BRANCH=dev-v2.8 (/package/Dockerfile, line 1)
* ARG CHART_DEFAULT_BRANCH=dev-v2.8 (/package/Dockerfile, line 1)
* ChartDefaultBranch = NewSetting("chart-default-branch", "dev-v2.8") (/pkg/settings/setting.go, line 116)
* SYSTEM_CHART_DEFAULT_BRANCH=${SYSTEM_CHART_DEFAULT_BRANCH:-"dev-v2.8"} (/scripts/package-env, line 5)
* CHART_DEFAULT_BRANCH=${CHART_DEFAULT_BRANCH:-"dev-v2.8"} (/scripts/package-env, line 7)
```

## Contributions

- File Issue with details of the problem, feature request, etc.
- Submit a pull request and include details of what problem or feature the code is solving or implementing.
