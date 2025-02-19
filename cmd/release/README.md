# release

Utility to do K3s and RKE2 releases.

### Examples

```sh
release -h
```

### K3s Release
#### Requirements
* OS: Linux, macOS
* Docker
* Git
* Go (At least the version used upstream for kubernetes)
* Sed (GNU for Linux or BSD for macOS)
* All commands require a Github token (classic) with the following permissions:
  * Be generated on behalf of an account with access to the `k3s-io/k3s` repo
  * `repo`
  * `write:packages`
* An SSH key, follow the Github [Documentation](https://docs.github.com/en/authentication/connecting-to-github-with-ssh) to generate one.
* A valid config file at `~/.ecm-distro-tools/config.json`

#### Commands
```bash
release generate k3s tags v1.29.2
release push k3s tags v1.29.2
release update k3s references v1.29.2
release tag k3s rc v1.29.2
release tag system-agent-installer-k3s rc v1.29.2
release tag k3s ga v1.29.2
release tag system-agent-installer-k3s ga v1.29.2
release stats -r rke2 -s 2024-01-01 -e 2024-12-31
release inspect v1.29.2+rke2r1
```

#### Cache Permissions and Docker:
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

### Rancher Release
#### Examples
##### List all RC and dev components in a git ref
Git ref can be a tag, branch, or commit hash.
```bash
release list rancher rc-deps release/v2.7
release list rancher rc-deps 8c7bbcaabcfabb00b1c89e55ed4f68117f938262
release list rancher rc-deps v2.7.12-rc1
```

### Charts Release
#### Examples
##### Default workflow

```bash
release list charts 2.9
release update charts 2.9 rancher-vsphere-csi 104.0.1+up3.3.0-rancher2
release push charts 2.9

# to inspect before pushing
release push charts 2.9 debug
```

Configure your autocompletion on `zsh` or `bash` for `release chart` commands.

#### `zsh` configuration example:
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

### Inspect Command Output
The `inspect` subcommand list information about images used by a published rke2 release.

```
$ release inspect v1.29.9-rc1+rke2r1

image                                            oss  prime  sig  amd64  arm64  win
-----                                            ---  -----  ---  -----  -----  ---
rancher/hardened-coredns:v1.12.0-build20241126   ✓    ✓      ?    ✓      ✓      -
rancher/mirrored-library-traefik:2.11.10         ✓    ✓      ?    ✓      ✓      -
```
