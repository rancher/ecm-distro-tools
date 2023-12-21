# k3s_release

k3s_release is a utility that performs the bulk of the actions to complete a K3s release.

Please reference the help menu from the binary.

## Requirements
* OS: Linux
* Docker
* Git
* Go (At least the version used upstream for kubernetes)
* Sed (GNU)
* All commands require a Github token (classic) with the following permissions:
  * Be generated on behalf of an account with access to the `k3s-io/k3s` repo
  * `repo`
  * `write:packages`    
* An SSH key is also required, follow the Github [Documentation](https://docs.github.com/en/authentication/connecting-to-github-with-ssh) to generate one.

## Configuration
| Name            | Description                                                                                                |
|-----------------|------------------------------------------------------------------------------------------------------------|
| old_k8s_version | Previous k8s patch version                                                                                 |
| new_k8s_version | Latest released k8s patch version                                                                          |
| old_k8s_client  | Previous k8s client patch version, usually the same as the k8s version, but with a major of 0 instead of 1 |
| new_k8s_client  | Latest released k8s client patch version                                                                   |
| old_k3s_version | Previous patch version suffix e.g: `k3s1`, this is used to update dependencies                             |
| new_k3s_version | Suffix for the next version `k3s1`                                                                         |
| release_branch  | Branch in `k3s-io/k3s` for the minor version e.g: `release-1.28`                                           |
| workspace       | Local directory to clone repos and create files                                                            |
| handler         | Your Github username                                                                                       |
| email           | Email to signoff commits                                                                                   |
| token           | Github Token described [above](#requirements)                                                              |
| ssh_key_path    | Path for the local private ssh key                                                                         |

Example:
```json
{
  "old_k8s_version": "v1.28.4",
  "new_k8s_version": "v1.28.5",
  "old_k8s_client": "v0.28.4",
  "new_k8s_client": "v0.28.5",
  "old_k3s_version": "k3s1",
  "new_k3s_version": "k3s1",
  "release_branch": "release-1.28",
  "workspace": "$GOPATH/src/github.com/kubernetes",
  "handler": "YourUsername",
  "email": "your.name@suse.com",
  "token": "${GITHUB_TOKEN}",
  "ssh_key_path": "$HOME/.ssh/github"
}
```

## Contributions

* File Issue with details of the problem, feature request, etc.
* Submit a pull request and include details of what problem or feature the code is solving or implementing.
