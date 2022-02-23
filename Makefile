.PHONY: all
all: gen-release-notes backport standup

.PHONY: gen-release-notes
gen-release-notes:
	cd cmd/$@ && $(MAKE)

.PHONY: backport
backport:
	cd cmd/$@ && $(MAKE)

.PHONY: standup
standup:
	cd cmd/$@ && $(MAKE)
