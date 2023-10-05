# upstream_go_version

The upstream_go_version prints out the version of Go used for a given branch of Kubernetes (upstream).

### Examples

```sh
upstream_go_version -b 'release-1.26'
release-1.26: 1.20.6
```

```sh
upstream_go_version -b 'release-1.25,release-1.26'
release-1.25: 1.20.6
release-1.26: 1.20.6
```

## Contributions

* File Issue with details of the problem, feature request, etc.
* Submit a pull request and include details of what problem or feature the code is solving or implementing.
