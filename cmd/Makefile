BINARIES = backport release
ARCHS ?= amd64 arm64
OSs ?= linux darwin
GO_COMPILE = GOOS=$${os} GOARCH=$${arch} CGO_ENABLED=1 $(GO) build -tags $(TAGS) -v -ldflags "$(LDFLAGS)" -o $@-$${os}-$${arch}
OS := $(shell uname)

ifeq ($(OS),Darwin)
SHA256SUM = shasum -a 256
else
SHA256SUM = sha256sum
endif

GEN_HASH = (cd $(BINDIR) && $(SHA256SUM) $(notdir $@)-$${os}-$${arch} >> sha256sums-$(BINARY).txt)
