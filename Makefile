.PHONY: all
all: gen-release-notes backport chart-up

.PHONY: gen-release-notes
gen-release-notes:
	cd cmd/$@ && $(MAKE)

.PHONY: backport
backport:
	cd cmd/$@ && $(MAKE)

.PHONY: chart-up
chart-up:
	cd cmd/$@ && $(MAKE)
