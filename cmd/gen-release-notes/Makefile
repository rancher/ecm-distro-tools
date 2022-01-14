GO = go

BINDIR := bin
BINARY := gen-release-notes

VERSION = v0.4.0
GIT_SHA = $(shell git rev-parse HEAD)
LDFLAGS = -ldflags "-X main.gitSHA=$(GIT_SHA) -X main.version=$(VERSION) -X main.name=$(BINARY)"
TAGS    = "netgo osusergo no_stage static_build"
$(BINDIR)/$(BINARY): clean
	CGO_ENABLED=1 $(GO) build -tags $(TAGS) -v $(LDFLAGS) -o $@

.PHONY: clean
clean:
	$(GO) clean
	rm -f $(BINDIR)/$(BINARY)

.PHONY: image
image: $(BINDIR)/$(BINARY)
	docker build -t briandowns/$(BINARY):$(VERSION) .
