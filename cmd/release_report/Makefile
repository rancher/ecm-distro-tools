CC = clang

UNAME_S = $(shell uname -s)

VERSION = 0.1.0
BINDIR  = bin
INCDIR  = include
BINARY  = rel-rep
CFLAGS  = -std=c99 -Wall -Wextra -fpic  \
          -Dbin_name=$(BINARY)          \
		  -Del_rep_version=$(VERSION) \
		  -Dgit_sha=$(shell git rev-parse HEAD)
LDFLAGS = -lcurl -lpthread -ljansson

PREFIX = /usr/local

$(BINDIR)/$(BINARY): $(BINDIR) clean
	$(CC) $(CFLAGS) main.c github.c -o $(BINDIR)/$(BINARY) $(LDFLAGS)
	
$(BINDIR):
	mkdir -p $(BINDIR)

.PHONY: clean
clean:
	rm -rf $(BINDIR)/*
