# backport

### Examples

```sh
# Backport K3s change into release-1.21 and release-1.22
./backport -r k3s -b 'release-1.21,release-1.22' -i 123 -c '181210f8f9c779c26da1d9b2075bde0127302ee0'

# Backport RKE2 change into release-1.20, release-1.21 and release-1.22
./backport -r rke2 -b 'release-1.20,release-1.21,release-1.22' -i 456 -c 'cd700d9a444df8f03b8ce88cb90261ed1bc49f27'
```

Or via Docker

```sh
docker run --rm -it rancher/ecm-distro-tools backport:v0.1.0 backport
```

## Contributions

* File Issue with details of the problem, feature request, etc.
* Submit a pull request and include details of what problem or feature the code is solving or implementing.
