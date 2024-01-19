# backport

The backport utility will create backport issues and perform a cherry-pick of the given commits to the given branches, if commits are provided on the CLI. If no commits are given, only the backport issues are created.

If a commit is provided, `backport` assumes you're running from the repository the operation is related to. This is simply to avoid having to guess or figure out where you store your code on your local system.

### Flags

| **Flag**                            | **Description**                                                                                                                                                                                      | **Required** |
| ----------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------ |
| `repo`, `r`                         | Name of the repository to perform the backport, e.g: `k3s`, `rke2`                                                                                                                                   | TRUE         |
| `issue`, `i`                        | ID of the original issue on GitHub                                                                                                                                                                   | TRUE         |
| `branches`, `b`                     | Branches the issue is being backported to, one or more (comma separated)                                                                                                                             | TRUE         |
| `owner`, `o`                        | Owner of the repository e.g: `k3s-io`, `rancher`                                                                                                                                                     | TRUE         |
| `commits`, `c`                      | Commits to be backported, if none is provided, only the issues will be created. When passing this flag, it assumes you're running from the repository this operation is related to (comma separated) | FALSE        |
| `user`, `u`                         | User to assign new issues to (default: user assignted to the original issue)                                                                                                                         | FALSE        |
| `dry-run`, `n`                      | Skip creating issues and pushing changes to remote                                                                                                                                                   | FALSE        |
| `skip-create-issue`, `s`            | Skip creating issues                                                                                                                                                                                 | FALSE        |
| `github-token`, `g`, `GITHUB_TOKEN` | Github Token                                                                                                                                                                                         | TRUE         |

### Examples

* Backport K3s change into release-1.21 and release-1.22. Only create the backport issues.
```sh
cd k3s
export GITHUB_TOKEN={YOUR_GITHUB_TOKEN}
backport -r k3s -o k3s-io -b 'release-1.21,release-1.22' -i 123
```

* Backport K3s change into release-1.21 and release-1.22. Creates the backport issues and cherry-picks the given commit id.
```sh
cd k3s
export GITHUB_TOKEN={YOUR_GITHUB_TOKEN}
backport -r k3s -o k3s-io -b 'release-1.21,release-1.22' -i 123 -c '181210f8f9c779c26da1d9b2075bde0127302ee0'
```

* Backport RKE2 change into release-1.20, release-1.21 and release-1.22
```sh
cd rke2
export GITHUB_TOKEN={YOUR_GITHUB_TOKEN}
backport -r rke2 -o rancher -b 'release-1.20,release-1.21,release-1.22' -i 456 -c 'cd700d9a444df8f03b8ce88cb90261ed1bc49f27'
```

* Backport K3s change into release-1.21 and release-1.22 and assign to given user.
```sh
cd k3s
export GITHUB_TOKEN={YOUR_GITHUB_TOKEN}
backport -r k3s -o k3s-io -b 'release-1.21,release-1.22' -i 123 -u susejsmith
```

## Contributions

* File Issue with details of the problem, feature request, etc.
* Submit a pull request and include details of what problem or feature the code is solving or implementing.
