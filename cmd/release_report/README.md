# Release Report

Generate a simple report that displays status and counts for a given tag

## Build

```sh
make
```

## Examples

```sh
rel-rep v1.33.1+k3s1
```

```sh
Tag:             v1.33.1+k3s1
Branch:          release-1.33
Pre-Release:     false
Assets:          16
```

```sh
rel-rep v1.33.1+rke2r1
```

```sh
Tag:             v1.33.1+rke2r1
Branch:          release-1.33
Pre-Release:     false
Assets:          74
RPMs .testing.0: 60
RPMs  .latest.0: 60
RPMs  .stable.0: 60
```

Multiple calls per execution

```sh
rel-rep v1.33.1+rke2r1,v1.29.13+rke2r1
```

```sh
Tag:             v1.33.1+rke2r1
Branch:          release-1.33
Pre-Release:     false
Assets:          74
RPMs .testing.0: 60
RPMs  .latest.0: 60
RPMs  .stable.0: 60

Tag:             v1.29.13+rke2r1
Branch:          release-1.29
Pre-Release:     false
Assets:          68
RPMs .testing.0: 60
RPMs  .latest.0: 60
RPMs  .stable.0: 60

```