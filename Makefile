.PHONY: all
all: gen_release_notes gen_release_report backport standup k3s_release rancher_release test_coverage upstream_go_version semv

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

.PHONY: semv
semv:
	cd cmd/$@ && $(MAKE)

.PHONY: test
test:
	go test -v -cover ./...

.PHONY: build-image
build-image:
	docker build -t rancher/ecm-distro-tools:$(shell git rev-parse HEAD) .
