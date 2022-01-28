.PHONY: all
all: gen-release-notes backport update-rke2-charts

.PHONY: gen-release-notes
gen-release-notes:
	cd cmd/$@ && $(MAKE)

.PHONY: backport
backport:
	cd cmd/$@ && $(MAKE)

.PHONY: update-rke2-charts
update-rke2-charts:
	cd cmd/$@ && $(MAKE)
