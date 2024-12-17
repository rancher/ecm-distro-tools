CC = cc

BINDIR = bin
BINARY = eoy_report

CFLAGS = -O3
ifdef DEBUG
	CFLAGS += -g
endif
LDFLAGS = -lcurl -ljansson -lpthread

$(BINDIR)/$(BINARY): $(BINDIR) clean
	$(CC) -o $@ main.c github.c spinner.c $(CFLAGS) $(LDFLAGS)

$(BINDIR):
	mkdir $@

.PHONY: clean
clean:
	rm -rf $(BINDIR)/$(BINARY)
