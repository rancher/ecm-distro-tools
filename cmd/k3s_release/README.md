# k3s_release

k3s_release is a utility that performs the bulk of the actions to complete a K3s release.

Please reference the help menu from the binary.


## Configuration
| Name             | Description                                                                                                |
| ---------------- | ---------------------------------------------------------------------------------------------------------- |
| old_k8s_version  | Previous k8s patch version                                                                                 |
| new_k8s_version  | Latest released k8s patch version                                                                          |
| old_k8s_client   | Previous k8s client patch version, usually the same as the k8s version, but with a major of 0 instead of 1 |
| new_k8s_client   | Latest released k8s client patch version                                                                   |
| old_k3s_suffix   | Previous patch version suffix e.g: `k3s1`, this is used to update dependencies                             |
| new_k3s_suffix   | Suffix for the next version `k3s1`                                                                         |
| release_branch   | Branch in `k3s-io/k3s` for the minor version e.g: `release-1.28`                                           |
| workspace        | Local directory to clone repos and create files                                                            |
| handler          | Your Github username                                                                                       |
| k3s_remote       | Custom K3S Remote, not required, defaults to `k3s-io`                                                      |
| k8s_rancher_url  | Custom K8s Fork URL, not required, defaults to `git@github.com:k3s-io/kubernetes.git`                      |
| k3s_upstream_url | Custom K3s Upstream URL, not required, defaults to `https://github.com/k3s-io/k3s`                         |
| email            | Email to signoff commits                                                                                   |
| ssh_key_path     | Path for the local private ssh key                                                                         |

Example:
```json
{
  "old_k8s_version": "v1.28.4",
  "new_k8s_version": "v1.28.5",
  "old_k8s_client": "v0.28.4",
  "new_k8s_client": "v0.28.5",
  "old_k3s_suffix": "k3s1",
  "new_k3s_suffix": "k3s1",
  "release_branch": "release-1.28",
  "workspace": "$GOPATH/src/github.com/kubernetes",
  "handler": "YourUsername",
  "email": "your.name@suse.com",
  "ssh_key_path": "$HOME/.ssh/github"
}
```
Export your Github token as an environment variable and run commands:
```bash
k3s_release push-tags -c config.json
k3s_release modify-k3s -c config.json
```

## Errors



## Contributions

* File Issue with details of the problem, feature request, etc.
* Submit a pull request and include details of what problem or feature the code is solving or implementing.
