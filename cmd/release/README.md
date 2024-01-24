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
* An SSH key is also required, follow the Github [Documentation](https://docs.github.com/en/authentication/connecting-to-github-with-ssh) to generate one.
* A valid config file at `~/.ecm-distro-tools/config.json`

#### Commands
```bash
# [...]
release k3s tag rc v1.29.2
release k3s tag ga v1.29.2
```


## Contributions

* File Issue with details of the problem, feature request, etc.
* Submit a pull request and include details of what problem or feature the code is solving or implementing.
