CC      ?= clang
DOCKER  ?= docker

VERSION = 0.1.0
BINDIR  = bin
INCDIR  = include
BINARY  = chart-compare
CFLAGS  = -std=c99 -Wall -Wextra -fpic \
          -Dbin_name=$(BINARY) \
          -Dcharts_compare_version=$(VERSION) \
		  -Dgit_sha=$(shell git rev-parse HEAD)
LDFLAGS = -lm -lcurl -lyaml -lrattler
PREFIX = /usr/local

MACOS_MANPAGE_LOC = /usr/share/man
LINUX_MAPPAGE_LOC = /usr/local/man/man8

$(BINDIR)/$(BINARY): $(BINDIR) clean
	$(CC) $(CFLAGS) main.c -o $(BINDIR)/$(BINARY) $(LDFLAGS)
	
$(BINDIR):
	mkdir -p $(BINDIR)

.PHONY: install
install: clean $(BINDIR)/$(BINARY)
	install $(BINDIR)/$(BINARY) $(PREFIX)/$(BINDIR)/$(BINARY)

.PHONY: uninstall
uninstall: 
	rm -f $(PREFIX)/$(BINDIR)/$(BINARY)*

#.PHONY: test
#test:
#	tests/tests
#	rm -f tests/tests

.PHONY: image
image:
	$(DOCKER) build -t $(BINARY):latest .

.PHONY: push
push:
	$(DOCKER) push briandowns/$(BINARY):latest

.PHONY: clean
clean:
	rm -f $(BINDIR)/*
