# Publish image action

This GitHub action aims to abstract the logic behind pushing container
images for projects within the Rancher ecosystem.
Depending on the project its images may need to be pushed to the Public
or Prime registry, in some cases both (default).
Use push-to-public and push-to-prime to pick the target registries.

## General guidance

* Transition the project to build images using Docker buildx

* The build process must be self-contained.
Ensure that the Dockerfile is responsible for as many of the steps as possible.
For example, it must compile the binaries of the project as a Docker layer,
instead of copying a pre-compiled binary from disk.

* Never push release candidates or any other pre-release versions to the Prime registry.

* The release workflow MUST be named `.github/workflows/release.yml`

## Makefile example

Notes:

* `TARGET_PLATFORMS` is set by the action based on the `platforms` input.
* `IID_FILE_FLAG` is set by the action and the make target **must** include this flag.

```make
MACHINE := rancher

# Define the target platforms that can be used across the ecosystem.
# Note that what would actually be used for a given project will be
# defined in TARGET_PLATFORMS, and must be a subset of the below:
DEFAULT_PLATFORMS := linux/amd64,linux/arm64,darwin/arm64,darwin/amd64

TAG ?= dev
REPO ?= rancher
IMAGE ?= ecm-distro-tools
IMAGE_NAME ?= $(REPO)/$(IMAGE):$(TAG)

buildx-machine: ## create rancher machine targeting platform defined by DEFAULT_PLATFORMS.
 @docker buildx ls | grep $(MACHINE) || \
  docker buildx create --name=$(MACHINE) --platform=$(DEFAULT_PLATFORMS)

.PHONY: push-image
push-image:
 docker buildx build \
  $(IID_FILE_FLAG) \
  $(BUILDX_ARGS) \
  --platform=$(TARGET_PLATFORMS) \
  --tag $(IMAGE_NAME) \
  --push \
  .

.PHONY: push-prime-image
push-prime-image:
  BUILDX_ARGS="--sbom=true --attest type=provenance,mode=max" \
  $(MAKE) push-image
```

## Action usage examples

### Release to public and prime

Recommended if you're building base images

Result images:

* `docker.io/rancher/ecm-distro-tools:v0.0.1` (linux/amd64,linux/arm64)
* `prime.registry/rancher/ecm-distro-tools:v0.0.1` (linux/amd64,linux/arm64)

```yml
name: Release
on:
  push:
    tags:
      - '*'
jobs:
  push-multiarch:
    permissions:
      contents: read
      id-token: write
    runs-on: runs-on,runner=8cpu-linux-x64,run-id=${{ github.run_id }},image=ubuntu22-full-x64
    steps:
    - name: Check out code
      uses: actions/checkout@v6

    - name: "Read secrets"
      uses: rancher-eio/read-vault-secrets@main
      with:
        secrets: |
          secret/data/github/repo/${{ github.repository }}/dockerhub/${{ github.repository_owner }}/credentials username | DOCKER_USERNAME ;
          secret/data/github/repo/${{ github.repository }}/dockerhub/${{ github.repository_owner }}/credentials password | DOCKER_PASSWORD ;
          secret/data/github/repo/${{ github.repository }}/rancher-prime-registry/credentials registry | PRIME_REGISTRY ;
          secret/data/github/repo/${{ github.repository }}/rancher-prime-registry/credentials username | PRIME_REGISTRY_USERNAME ;
          secret/data/github/repo/${{ github.repository }}/rancher-prime-registry/credentials password | PRIME_REGISTRY_PASSWORD ;

    - name: Push image to public
      uses: rancher/ecm-distro-tools/actions/publish-image@<commit-sha>
      with:
        image: ecm-distro-tools
        tag: ${{ github.ref_name }}
        platforms: linux/amd64,linux/arm64
        push-to-prime: false

        public-repo: rancher
        public-username: ${{ env.DOCKER_USERNAME }}
        public-password: ${{ env.DOCKER_PASSWORD }}
        make-target: push-image


    - name: Push image to prime
      if: ${{ !contains(github.ref_name, '-rc') }} # never push pre-release images to prime
      uses: rancher/ecm-distro-tools/actions/publish-image@<commit-sha>
      with:
        image: ecm-distro-tools
        tag: ${{ github.ref_name }}
        platforms: linux/amd64,linux/arm64
        push-to-public: false

        prime-repo: rancher
        prime-registry: ${{ env.PRIME_REGISTRY }}
        prime-username: ${{ env.PRIME_REGISTRY_USERNAME }}
        prime-password: ${{ env.PRIME_REGISTRY_PASSWORD }}
        prime-make-target: push-prime-image
```

### Release to public and staging

Recommended if you're building images for charts or rancher that will be at `rancher-images.txt`

Note: the `identity-registry` input is recommended here to ensure that the pull
reference will match the signature after syncing the image from staging to prime.

Result images:

* `docker.io/rancher/ecm-distro-tools:v0.0.1` (linux/amd64,linux/arm64)
* `staging.registry/rancher/ecm-distro-tools:v0.0.1` (linux/amd64,linux/arm64)

```yml
name: Release
on:
  push:
    tags:
      - '*'
jobs:
  push-multiarch:
    permissions:
      contents: read
      id-token: write
    runs-on: runs-on,runner=8cpu-linux-x64,run-id=${{ github.run_id }},image=ubuntu22-full-x64
    steps:
    - name: Check out code
      uses: actions/checkout@v6

    - name: "Read secrets"
      uses: rancher-eio/read-vault-secrets@main
      with:
        secrets: |
          secret/data/github/repo/${{ github.repository }}/dockerhub/${{ github.repository_owner }}/credentials username | DOCKER_USERNAME ;
          secret/data/github/repo/${{ github.repository }}/dockerhub/${{ github.repository_owner }}/credentials password | DOCKER_PASSWORD ;
          secret/data/github/repo/${{ github.repository }}/rancher-prime-registry/credentials registry | PRIME_REGISTRY ;
          secret/data/github/repo/${{ github.repository }}/rancher-prime-staging-registry/credentials registry | PRIME_STG_REGISTRY ;
          secret/data/github/repo/${{ github.repository }}/rancher-prime-staging-registry/credentials username | PRIME_STG_REGISTRY_USERNAME ;
          secret/data/github/repo/${{ github.repository }}/rancher-prime-staging-registry/credentials password | PRIME_STG_REGISTRY_PASSWORD ;

    - name: Push images
      uses: rancher/ecm-distro-tools/actions/publish-image@<commit-sha>
      with:
        image: ecm-distro-tools
        tag: ${{ github.ref_name }}
        platforms: linux/amd64,linux/arm64

        public-repo: rancher
        public-username: ${{ env.DOCKER_USERNAME }}
        public-password: ${{ env.DOCKER_PASSWORD }}
        make-target: push-image

        prime-repo: rancher
        identity-registry: ${{ env.PRIME_REGISTRY }}
        prime-registry: ${{ env.PRIME_STG_REGISTRY }}
        prime-username: ${{ env.PRIME_STG_REGISTRY_USERNAME }}
        prime-password: ${{ env.PRIME_STG_REGISTRY_PASSWORD }}
        prime-make-target: push-prime-image
```

### Release only to public

Result images:

* `docker.io/rancher/ecm-distro-tools:v0.0.1` (linux/amd64,linux/arm64)

```yml
name: Release
on:
  push:
    tags:
      - '*'
jobs:
  push-multiarch:
    permissions:
      contents: read
      id-token: write
    runs-on: runs-on,runner=8cpu-linux-x64,run-id=${{ github.run_id }},image=ubuntu22-full-x64
    steps:
    - name: Check out code
      uses: actions/checkout@v6

    - name: "Read secrets"
      uses: rancher-eio/read-vault-secrets@main
      with:
        secrets: |
          secret/data/github/repo/${{ github.repository }}/dockerhub/${{ github.repository_owner }}/credentials username | DOCKER_USERNAME ;
          secret/data/github/repo/${{ github.repository }}/dockerhub/${{ github.repository_owner }}/credentials password | DOCKER_PASSWORD ;

    - name: Push images
      uses: rancher/ecm-distro-tools/actions/publish-image@<commit-sha>
      with:
        image: ecm-distro-tools
        tag: ${{ github.ref_name }}
        platforms: linux/amd64,linux/arm64

        public-repo: rancher
        public-username: ${{ env.DOCKER_USERNAME }}
        public-password: ${{ env.DOCKER_PASSWORD }}
        make-target: push-image
        push-to-prime: false
```

### Release to prime and staging

Recommended if you're building images that won't be public

Result images:

* `staging.registry/rancher/ecm-distro-tools:v0.0.1` (linux/amd64,linux/arm64)
* `prime.registry/rancher/ecm-distro-tools:v0.0.1` (linux/amd64,linux/arm64)

```yml
name: Release
on:
  push:
    tags:
      - '*'
jobs:
  push-multiarch:
    permissions:
      contents: read
      id-token: write
    runs-on: runs-on,runner=8cpu-linux-x64,run-id=${{ github.run_id }},image=ubuntu22-full-x64
    steps:
    - name: Check out code
      uses: actions/checkout@v6

    - name: "Read secrets"
      uses: rancher-eio/read-vault-secrets@main
      with:
        secrets: |
          secret/data/github/repo/${{ github.repository }}/rancher-prime-staging-registry/credentials registry | PRIME_STG_REGISTRY ;
          secret/data/github/repo/${{ github.repository }}/rancher-prime-staging-registry/credentials username | PRIME_STG_REGISTRY_USERNAME ;
          secret/data/github/repo/${{ github.repository }}/rancher-prime-staging-registry/credentials password | PRIME_STG_REGISTRY_PASSWORD ;
          secret/data/github/repo/${{ github.repository }}/rancher-prime-registry/credentials registry | PRIME_REGISTRY ;
          secret/data/github/repo/${{ github.repository }}/rancher-prime-registry/credentials username | PRIME_REGISTRY_USERNAME ;
          secret/data/github/repo/${{ github.repository }}/rancher-prime-registry/credentials password | PRIME_REGISTRY_PASSWORD ;

    - name: Push image to staging
      uses: rancher/ecm-distro-tools/actions/publish-image@<commit-sha>
      with:
        image: ecm-distro-tools
        tag: ${{ github.ref_name }}
        platforms: linux/amd64,linux/arm64
        push-to-public: false

        prime-repo: rancher
        prime-registry: ${{ env.PRIME_STG_REGISTRY }}
        prime-username: ${{ env.PRIME_STG_REGISTRY_USERNAME }}
        prime-password: ${{ env.PRIME_STG_REGISTRY_PASSWORD }}
        prime-make-target: push-prime-image


    - name: Push image to prime
      if: ${{ !contains(github.ref_name, '-rc') }} # never push pre-release images to prime
      uses: rancher/ecm-distro-tools/actions/publish-image@<commit-sha>
      with:
        image: ecm-distro-tools
        tag: ${{ github.ref_name }}
        platforms: linux/amd64,linux/arm64
        push-to-public: false

        prime-repo: rancher
        prime-registry: ${{ env.PRIME_REGISTRY }}
        prime-username: ${{ env.PRIME_REGISTRY_USERNAME }}
        prime-password: ${{ env.PRIME_REGISTRY_PASSWORD }}
        prime-make-target: push-prime-image
```

### Release to public, prime and staging

Result images:

* `docker.io/rancher/ecm-distro-tools:v0.0.1` (linux/amd64,linux/arm64)
* `staging.registry/rancher/ecm-distro-tools:v0.0.1` (linux/amd64,linux/arm64)
* `prime.registry/rancher/ecm-distro-tools:v0.0.1` (linux/amd64,linux/arm64)

```yml
name: Release
on:
  push:
    tags:
      - '*'
jobs:
  push-multiarch:
    permissions:
      contents: read
      id-token: write
    runs-on: runs-on,runner=8cpu-linux-x64,run-id=${{ github.run_id }},image=ubuntu22-full-x64
    steps:
    - name: Check out code
      uses: actions/checkout@v6

    - name: "Read secrets"
      uses: rancher-eio/read-vault-secrets@main
      with:
        secrets: |
          secret/data/github/repo/${{ github.repository }}/dockerhub/${{ github.repository_owner }}/credentials username | DOCKER_USERNAME ;
          secret/data/github/repo/${{ github.repository }}/dockerhub/${{ github.repository_owner }}/credentials password | DOCKER_PASSWORD ;
          secret/data/github/repo/${{ github.repository }}/rancher-prime-staging-registry/credentials registry | PRIME_STG_REGISTRY ;
          secret/data/github/repo/${{ github.repository }}/rancher-prime-staging-registry/credentials username | PRIME_STG_REGISTRY_USERNAME ;
          secret/data/github/repo/${{ github.repository }}/rancher-prime-staging-registry/credentials password | PRIME_STG_REGISTRY_PASSWORD ;
          secret/data/github/repo/${{ github.repository }}/rancher-prime-registry/credentials registry | PRIME_REGISTRY ;
          secret/data/github/repo/${{ github.repository }}/rancher-prime-registry/credentials username | PRIME_REGISTRY_USERNAME ;
          secret/data/github/repo/${{ github.repository }}/rancher-prime-registry/credentials password | PRIME_REGISTRY_PASSWORD ;

    - name: Push image to public and staging
      uses: rancher/ecm-distro-tools/actions/publish-image@<commit-sha>
      with:
        image: ecm-distro-tools
        tag: ${{ github.ref_name }}
        platforms: linux/amd64,linux/arm64

        public-repo: rancher
        public-username: ${{ env.DOCKER_USERNAME }}
        public-password: ${{ env.DOCKER_PASSWORD }}
        make-target: push-image

        prime-repo: rancher
        identity-registry: ${{ env.PRIME_REGISTRY }}
        prime-registry: ${{ env.PRIME_STG_REGISTRY }}
        prime-username: ${{ env.PRIME_STG_REGISTRY_USERNAME }}
        prime-password: ${{ env.PRIME_STG_REGISTRY_PASSWORD }}
        prime-make-target: push-prime-image

    - name: Push image to prime
      if: ${{ !contains(github.ref_name, '-rc') }} # never push pre-release images to prime
      uses: rancher/ecm-distro-tools/actions/publish-image@<commit-sha>
      with:
        image: ecm-distro-tools
        tag: ${{ github.ref_name }}
        platforms: linux/amd64,linux/arm64
        push-to-public: false

        prime-repo: rancher
        prime-registry: ${{ env.PRIME_REGISTRY }}
        prime-username: ${{ env.PRIME_REGISTRY_USERNAME }}
        prime-password: ${{ env.PRIME_REGISTRY_PASSWORD }}
        prime-make-target: push-prime-image
```

### Release only to prime using the directory input

It's not recommended to push images directly to the prime registry without
validating release candidates on staging.

Note: the `Makefile` is located at the `build` directory

Result images:

* `prime.registry/rancher/ecm-distro-tools:v0.0.1` (linux/amd64,linux/arm64)

```yml
name: Release
on:
  push:
    tags:
      - '*'
jobs:
  push-multiarch:
    permissions:
      contents: read
      id-token: write
    runs-on: runs-on,runner=8cpu-linux-x64,run-id=${{ github.run_id }},image=ubuntu22-full-x64
    steps:
    - name: Check out code
      uses: actions/checkout@v6

    - name: "Read secrets"
      uses: rancher-eio/read-vault-secrets@main
      with:
        secrets: |
          secret/data/github/repo/${{ github.repository }}/rancher-prime-registry/credentials registry | PRIME_REGISTRY ;
          secret/data/github/repo/${{ github.repository }}/rancher-prime-registry/credentials username | PRIME_REGISTRY_USERNAME ;
          secret/data/github/repo/${{ github.repository }}/rancher-prime-registry/credentials password | PRIME_REGISTRY_PASSWORD ;

    - name: Push image to prime
      uses: rancher/ecm-distro-tools/actions/publish-image@<commit-sha>
      with:
        image: ecm-distro-tools
        tag: ${{ github.ref_name }}
        platforms: linux/amd64,linux/arm64
        push-to-public: false

        prime-repo: rancher
        prime-registry: ${{ env.PRIME_REGISTRY }}
        prime-username: ${{ env.PRIME_REGISTRY_USERNAME }}
        prime-password: ${{ env.PRIME_REGISTRY_PASSWORD }}
        prime-make-target: push-prime-image
        prime-make-target-directory: build
```

## Developing on forks

The GHA rancher/ecm-distro-tools/actions/publish-image can be used and tested on
forks, as it will use whatever input data that is provided to it.
To test it on a fork simply create your secrets within the GitHub fork and pass
them on as per GH syntax: prime-username: ${{ secrets.PRIME_REGISTRY_USERNAME }}.

When used in combination with rancher-eio/read-vault-secrets@main forks are no
longer supported, as read-vault-secrets requires a valid HashiCorp Vault
instance in order for it to work. In this case, the recommended approach is to
cut release candidate versions to test the end-to-end process.
