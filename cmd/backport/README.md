# backport

The backport utility will create backport issues and perform a cherry-pick of the given commits to the given branches, if commits are provided on the CLI. If no commits are given, only the backport issues are created.

If a commit is provided, `backport` assumes you're running from the repository the operation is related to. This is simply to avoid having to guess or figure out where you store your code on your local system.

### Examples

```sh
# Backport K3s change into release-1.21 and release-1.22. Only create the backport issues.
./backport -r k3s -b 'release-1.21,release-1.22' -i 123

# Backport K3s change into release-1.21 and release-1.22. Creates the backport issues and cherry-picks the given commit id.
./backport -r k3s -b 'release-1.21,release-1.22' -i 123 -c '181210f8f9c779c26da1d9b2075bde0127302ee0'

# Backport RKE2 change into release-1.20, release-1.21 and release-1.22
./backport -r rke2 -b 'release-1.20,release-1.21,release-1.22' -i 456 -c 'cd700d9a444df8f03b8ce88cb90261ed1bc49f27'

# Backport K3s change into release-1.21 and release-1.22 and assign to given user.
./backport -r k3s -b 'release-1.21,release-1.22' -i 123 -u susejsmith
```

## Contributions

* File Issue with details of the problem, feature request, etc.
* Submit a pull request and include details of what problem or feature the code is solving or implementing.
