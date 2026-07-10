# Follow the same pattern as other tools in cmd/
BINARY = prbuilder

include ../Makefile

.PHONY: build
build: $(BINARY)

$(BINARY):
	@echo "Building $(BINARY)..."
	@for os in $(OSs); do \
		for arch in $(ARCHS); do \
			echo "  Building $(BINARY)-$${os}-$${arch}..."; \
			$(GO_COMPILE); \
			$(GEN_HASH); \
		done; \
	done
