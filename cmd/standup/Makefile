VERSION          := v0.1.0
BINDIR           := bin
BINARY           := standup
override LDFLAGS +=
override CFLAGS  += -Dbin_name=$(BINARY) -Dstandup_version=$(VERSION) -Dgit_sha=$(shell git rev-parse HEAD) -O3

PREFIX = /usr/local

UNAME_S = $(shell uname -s)

MACOS_MANPAGE_LOC = /usr/share/man
LINUX_MAPPAGE_LOC = $(PREFIX)/man/man8

$(BINDIR)/$(BINARY): $(BINDIR) clean
ifeq ($(UNAME_S),FreeBSD)
	$(CC) main.c $(CFLAGS) -o $@-freebsd-amd64 $(LDFLAGS) -I$(PREFIX)/include -L$(PREFIX)/lib
else ifeq ($(UNAME_S),Linux)
	$(CC) main.c -static $(CFLAGS) -o $@-linux-amd64 $(LDFLAGS)
else
	$(CC) main.c $(CFLAGS) -o $@ $(LDFLAGS)
endif

$(BINDIR):
	mkdir -p $@

$(DEPDIR):
	mkdir -p $@

.PHONY: install
install: $(BINDIR)/$(BINARY)
	mkdir -p $(PREFIX)/bin
	cp $(BINDIR)/$(BINARY) $(PREFIX)/bin

.PHONY: uninstall
uninstall:
	rm -f $(PREFIX)/$(BINDIR)/$(BINARY)

.PHONY: deps
deps: $(DEPDIR)

.PHONY: clean
clean:
	rm -f $(BINDIR)/*
