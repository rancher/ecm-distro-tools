BINARIES = gen_release_notes \
	       gen_release_report \
		   backport \
		   semv \
		   rancher_release \
		   test_coverage \
		   upstream_go_version \
		   rke2_release release \
		   drone_migrator

ARCHS ?= amd64 arm64
OSs   ?= linux darwin

GO_COMPILE = GOOS=$${os} GOARCH=$${arch} CGO_ENABLED=1 $(GO) build -tags $(TAGS) -v -ldflags "$(LDFLAGS)" -o $@-$${os}-$${arch}
OS := $(shell uname)

ifeq ($(OS),Darwin)
GEN_HASH = shasum -a 256 $@-$${os}-$${arch} >> $(BINDIR)/sha256sums-$(BINARY).txt
else
GEN_HASH = sha256sum $@-$${os}-$${arch} >> $(BINDIR)/sha256sums-$(BINARY).txt
endif
