.PHONY: install uninstall deps clean

# We are on windows
ifdef OS
   EXT = .exe
endif

VERSION          := v0.1.0
BINDIR           := bin
BINARY           := standup$(EXT)
override LDFLAGS +=
override CFLAGS  += -Dbin_name=$(BINARY) -Dstandup_version=$(VERSION) -Dgit_sha=$(shell git rev-parse HEAD) -O3

PREFIX = /usr/local

UNAME_S = $(shell uname -s)

MACOS_MANPAGE_LOC = /usr/share/man
LINUX_MAPPAGE_LOC = $(PREFIX)/man/man8

$(BINDIR)/$(BINARY): $(BINDIR) main.c
ifeq ($(UNAME_S),FreeBSD)
	$(CC) main.c $(CFLAGS) -o $@ $(LDFLAGS) -I$(PREFIX)/include -L$(PREFIX)/lib
else ifeq ($(UNAME_S),Linux)
	$(CC) main.c -static $(CFLAGS) -o $@ $(LDFLAGS)
else
	$(CC) main.c $(CFLAGS) -o $@ $(LDFLAGS)
endif

$(BINDIR):
	mkdir -p $@

$(DEPDIR):
	mkdir -p $@

install: $(BINDIR)/$(BINARY)
	mkdir -p $(PREFIX)/bin
	cp $(BINDIR)/$(BINARY) $(PREFIX)/bin

uninstall:
	rm -f $(PREFIX)/$(BINDIR)/$(BINARY)

deps: $(DEPDIR)

clean:
	rm -f $(BINDIR)/*
