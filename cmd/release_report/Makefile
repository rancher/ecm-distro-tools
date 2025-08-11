CC = clang

UNAME_S = $(shell uname -s)

VERSION = 0.1.0
BINDIR  = bin
INCDIR  = include
BINARY  = rel-rep
CFLAGS  = -std=c17 -Wall -Wextra -fpic  \
          -Dbin_name=$(BINARY)          \
		  -Drel_rep_version=$(VERSION) \
		  -Dgit_sha=$(shell git rev-parse HEAD)
LDFLAGS = -lpthread
ifeq ($(UNAME_S),Darwin)
	LDFLAGS += $(shell pkg-config jansson --libs)
	LDFLAGS += $(shell pkg-config libcurl --libs)
	CFLAGS  += $(shell pkg-config jansson --cflags)
	CFLAGS  += $(shell pkg-config libcurl --cflags)
else
	LDFLAGS += -lcurl -ljansson
endif

PREFIX = /usr/local

$(BINDIR)/$(BINARY): $(BINDIR) clean
	$(CC) $(CFLAGS) main.c github.c -o $(BINDIR)/$(BINARY) $(LDFLAGS)
	
$(BINDIR):
	mkdir -p $(BINDIR)

.PHONY: clean
clean:
	rm -rf $(BINDIR)/*
