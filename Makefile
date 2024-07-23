include cmd/Makefile

ACTION ?= --load
MACHINE := rancher
# Define the target platforms that can be used across the ecosystem.
# Note that what would actually be used for a given project will be
# defined in TARGET_PLATFORMS, and must be a subset of the below:
DEFAULT_PLATFORMS := linux/amd64,linux/arm64
BUILDX_ARGS ?= --sbom=true --attest type=provenance,mode=max

ifeq ($(TAG),)
	TAG = $(shell git rev-parse HEAD)
endif

.PHONY: all
all: $(BINARIES)

.PHONY: gen_release_report
gen_release_report:
	cd cmd/$@ && $(MAKE)

.PHONY: rancher_release
rancher_release:
	cd cmd/$@ && $(MAKE)

.PHONY: rke2_release
rke2_release:
	cd cmd/$@ && $(MAKE)

.PHONY: release
release:
	cd cmd/$@ && $(MAKE)

.PHONY: backport
backport:
	cd cmd/$@ && $(MAKE)

.PHONY: test_coverage
test_coverage:
	cd cmd/$@ && $(MAKE)

.PHONY: upstream_go_version
upstream_go_version:
	cd cmd/$@ && $(MAKE)

.PHONY: semv
semv:
	cd cmd/$@ && $(MAKE)

.PHONY: test
test:
	go test -v -cover ./...

.PHONY: buildx-machine
buildx-machine: ## create rancher dockerbuildx machine targeting platform defined by DEFAULT_PLATFORMS.
	@docker buildx ls | grep $(MACHINE) || docker buildx create --name=$(MACHINE) --platform=$(DEFAULT_PLATFORMS)

.PHONY: build-image 
build-image: buildx-machine
	docker buildx build --builder=$(MACHINE) $(ACTION) -t rancher/ecm-distro-tools:$(TAG) .

.PHONY: push-image
push-image: buildx-machine ## build the container image targeting all platforms defined by TARGET_PLATFORMS and push to a registry.
	docker buildx build -f Dockerfile \
		--builder=$(MACHINE) $(IID_FILE_FLAG) $(BUILDX_ARGS) \
		--platform=$(DEFAULT_PLATFORMS) -t $(REPO):$(TAG) --push .
	@echo "Pushed $(REPO):$(TAG)"

.PHONY: test-image
test-image: buildx-machine ## build the container image for all target architecures.
	# Instead of loading image, target all platforms, effectivelly testing
	# the build for the target architectures.
	$(MAKE) build-image ACTION="--platform=$(DEFAULT_PLATFORMS)"

.PHONY: package-binaries
package-binaries: $(BINARIES)
	@$(eval export BIN_FILES = $(shell ls bin/))

	for arch in $(ARCHS); do \
		for os in $(OSs); do \
			SUFFIX=$${os}-$${arch}; \
			cd bin && \
			tar cvf ../ecm-distro-tools.$${SUFFIX}.tar $(BIN_FILES) && \
			cd ../; \
			for binary in $(BINARIES); do \
				cd cmd/$${binary}/bin && \
				tar rvf ../../../ecm-distro-tools.$${SUFFIX}.tar $${binary}-$${SUFFIX} && \
				cd ../../../; \
			done; \
			gzip < ecm-distro-tools.$${SUFFIX}.tar > ecm-distro-tools.$${SUFFIX}.tar.gz && \
			rm -f ecm-distro-tools.$${SUFFIX}.tar; \
		done; \
	done
