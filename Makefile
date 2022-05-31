.PHONY: all
all: gen-release-notes backport standup k3s-release

.PHONY: gen-release-notes
gen-release-notes:
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
