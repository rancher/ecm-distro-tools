# release

CLI for performing release tasks and reporting release status for Rancher and Rancher charts, K3s, and RKE2.

## Setup

Create a config file in the default location, ~/.ecm-distro-tools/config.json

```sh
release config gen > ~/.ecm-distro-tools/config.json
```

Show help

```sh
release -h
```

## K3s release

### Extra requirements
* Docker
* Git
* Go
* Github token (classic) with the following permissions:
  * Be generated on behalf of an account with access to the `k3s-io/k3s` repo
  * `repo`
  * `write:packages`
* An SSH key, follow the Github [Documentation](https://docs.github.com/en/authentication/connecting-to-github-with-ssh) to generate one.

### Commands
```bash
release generate k3s tags v1.29.2
release push k3s tags v1.29.2
release update k3s references v1.29.2
release tag k3s rc v1.29.2
release tag system-agent-installer-k3s rc v1.29.2
release tag k3s ga v1.29.2
release tag system-agent-installer-k3s ga v1.29.2
release generate k3s release notes \
  --prev-milestone v1.29.1+k3s1 \
  --milestone v1.29.2-rc1+k3s1
```

For new minor releases, use a commit SHA for `--prev-milestone` to begin after the last Kubernetes bump.

```sh
git log -G 'k8s.io/kubernetes' --since='1 month ago' -- go.mod # last k8s bump in main branch
release generate k3s release notes --prev-milestone 5411cbd3 --milestone v1.29.2-rc1+k3s1
```

### Cache Permissions and Docker
```bash
$ release generate k3s tags v1.26.12
> failed to rebase and create tags: chown: changing ownership of '/home/go/.cache': Operation not permitted
failed to initialize build cache at /home/go/.cache: mkdir /home/go/.cache/00: permission denied
```
Verify if the `$GOPATH/.cache` directory is owned by the same user that is running the command. If not, change the ownership of the directory:
```bash
$ ls -la $GOPATH/
> drwxr-xr-x  2 root root 4096 Dec 20 15:50 .cache
$ sudo chown $USER $GOPATH/.cache
```

## RKE2 release

Commands

```sh
release tag rke2 rc v1.29.2
release tag rke2 ga v1.29.2
release inspect v1.29.2+rke2r1
release stats -r rke2 -s 2024-01-01 -e 2024-12-31
release generate rke2 release notes \
  --prev-milestone v1.29.1+rke2r1 \
  --milestone v1.29.2-rc1+rke2r1
```

For new minor releases, use a commit SHA for `--prev-milestone` to begin after the last Kubernetes bump:

```sh
git log -G 'KUBERNETES_VERSION' --since='1 month ago' -- Dockerfile # last k8s bump in main branch
release generate rke2 release notes --prev-milestone 5411cbd3 --milestone v1.29.2-rc1+rke2r1
```

## Image build

Commands intended to be run in GitHub Actions workflows, not for CLI use.
See [sync-upstream-release action](../../actions/sync-upstream-release/action.yml)

```sh
release sync image-build \
  --image-build-repo <repo> \
  --image-build-owner <owner> \
  --upstream-repo <upstream> \
  --upstream-owner <owner> \
  --tag-prefix <prefix>
release sync republish-latest \
  --repo <repo> \
  --owner <owner> \
  --commitish <branch/sha>
```

## Rancher release

List all RC and dev components in a git ref.

```sh
release list rancher rc-deps release/v2.7
release list rancher rc-deps 8c7bbcaabcfabb00b1c89e55ed4f68117f938262
release list rancher rc-deps v2.7.12-rc1
```

Dashboard and UI releases. The release candidate number is automatically incremented.

```sh
release tag dashboard rc v2.9.0
release tag dashboard ga v2.9.0
release tag ui rc v2.9.0
release tag ui ga v2.9.0
```

## Charts Release

```sh
release list charts 2.9
release update charts 2.9 rancher-vsphere-csi 104.0.1+up3.3.0-rancher2
release push charts 2.9

# to inspect before pushing
release push charts 2.9 debug
```

## Completions

`release` provides completions for multiple shells.

### `zsh` configuration example:
```
./release completion zsh > completion.zsh
mv completion.zsh ~/.zsh/completion/completion.zsh
chmod +x ~/.zsh/completion/completion.zsh
```

Your `.zshrc` file must have something like the following:

```
# ECM-DISTRO-TOOLS
export PATH=$PATH:/<home_path>/.local/bin/ecm-distro-tools
source ~/.zsh/completion/completion.zsh
fpath=(~/.zsh/completion $fpath)
autoload -Uz compinit && compinit
```

## Contributions

* File Issue with details of the problem, feature request, etc.
* Submit a pull request and include details of what problem or feature the code is solving or implementing.
