.PHONY: all
all: gen-release-notes gen_release_report backport standup k3s-release

.PHONY: gen-release-notes
gen-release-notes:
	cd cmd/$@ && $(MAKE)

.PHONY: gen_release_report
gen_release_report:
	cd cmd/$@ && $(MAKE)

.PHONY: k3s-release
k3s-release:
	cd cmd/$@ && $(MAKE)

.PHONY: backport
backport:
	cd cmd/$@ && $(MAKE)

.PHONY: standup
standup:
	cd cmd/$@ && $(MAKE)

.PHONY: build-image
build-image:
	docker build -t ecmdt-local .
