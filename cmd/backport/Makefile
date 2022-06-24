.PHONY: clean

# We are on windows
ifdef OS
   EXT = .exe
endif

BINDIR := bin
BINARY := backport$(EXT)

GO      = go
VERSION = v0.1.0
GIT_SHA = $(shell git rev-parse HEAD)
LDFLAGS = -ldflags "-X main.gitSHA=$(GIT_SHA) -X main.version=$(VERSION) -X main.name=$(BINARY)"
TAGS    = "netgo osusergo no_stage static_build"

$(BINDIR)/$(BINARY): $(BINDIR) main.go
	CGO_ENABLED=1 $(GO) build -tags $(TAGS) -v $(LDFLAGS) -o $@

$(BINDIR):
	mkdir -p $@

clean:
	$(GO) clean
	rm -f $(BINDIR)/$(BINARY)
