# semv

`semv` provides subcommands to parse a semantic version, and to test if a version conforms to a semantic version constraint.

### Examples

```sh
PRERELEASE=$(./bin/semv-darwin-amd64 parse --output go-template="{{ .Prerelease }}" v1.2.3)
if [ -n "$PRERELEASE" ]; then
  echo "Prerelease: $PRERELEASE"
fi

```

```sh
semv test v1.1 v1.1.1
```

## Contributions

* File Issue with details of the problem, feature request, etc.
* Submit a pull request and include details of what problem or feature the code is solving or implementing.
