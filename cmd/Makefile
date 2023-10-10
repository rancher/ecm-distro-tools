BINARIES = gen_release_notes gen_release_report backport standup k3s_release rancher_release test_coverage upstream_go_version
ARCHS = amd64 arm64
OSs = linux darwin freebsd

GO_COMPILE = CGO_ENABLED=1 $(GO) build -tags $(TAGS) -v -ldflags "$(LDFLAGS)" -o $@-$${os}-$${arch}
