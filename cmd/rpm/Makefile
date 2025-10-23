include ../Makefile

GO = go

BINDIR := bin
BINARY := rpm

GIT_SHA           = $(shell git rev-parse HEAD)
VERSION           ?= snapshot-${GIT_SHA}
override LDFLAGS += -X main.gitSHA=$(GIT_SHA) -X main.version=$(VERSION) -X main.name=$(BINARY) -extldflags '-static -Wl,--fatal-warnings'
TAGS              = "netgo osusergo no_stage static_build"
$(BINDIR)/$(BINARY): clean
	for arch in $(ARCHS); do \
		for os in $(OSs); do \
			$(GO_COMPILE) ; \
			$(GEN_HASH); \
		done; \
	done

.PHONY: clean
clean:
	$(GO) clean
	rm -f $(BINDIR)/$(BINARY)

.PHONY: image
image: $(BINDIR)/$(BINARY)
	docker build -t rancher/$(BINARY):$(VERSION) .
