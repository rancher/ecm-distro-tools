include cmd/Makefile

.PHONY: all
all: $(BINARIES)

.PHONY: gen_release_notes
gen_release_notes:
	cd cmd/$@ && $(MAKE)

.PHONY: gen_release_report
gen_release_report:
	cd cmd/$@ && $(MAKE)

.PHONY: k3s_release
k3s_release:
	cd cmd/$@ && $(MAKE)

.PHONY: rancher_release
rancher_release:
	cd cmd/$@ && $(MAKE)

.PHONY: rke2_release
rke2_release:
	cd cmd/$@ && $(MAKE)

.PHONY: backport
backport:
	cd cmd/$@ && $(MAKE)

.PHONY: standup
standup:
	cd cmd/$@ && $(MAKE)

.PHONY: test_coverage
test_coverage:
	cd cmd/$@ && $(MAKE)

.PHONY: upstream_go_version
upstream_go_version:
	cd cmd/$@ && $(MAKE)

.PHONY: test
test:
	go test -v -cover ./...

.PHONY: build-image
build-image:
	docker build -t rancher/ecm-distro-tools:$(shell git rev-parse HEAD) .

.PHONY: package-binaries
package-binaries: # add dependency
	@$(eval export BIN_FILES = $(shell ls bin/))

	cd bin                                       && \
	tar cvf ../ecm-distro-tools.tar $(BIN_FILES) && \
	cd ../

	for binary in $(BINARIES); do \
		cd cmd/$${binary}/bin                   && \
		tar rvf ../../../ecm-distro-tools.tar * && \
		cd ../../../;                              \
	done

	gzip < ecm-distro-tools.tar > ecm-distro-tools.tgz && \
	rm -f ecm-distro-tools.tar
