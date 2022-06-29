.PHONY: clearscr fresh clean build-image all

# We are on windows
ifdef OS
   EXT = .exe
endif

COMMANDS := \
	cmd/gen-release-notes/bin/gen-release-notes$(EXT) \
	cmd/backport/bin/backport$(EXT) \
	cmd/standup/bin/standup$(EXT) \
	cmd/k3s-release/bin/k3s-release$(EXT)

all: $(COMMANDS)
	@true

%:
	mkdir -p $(@D)
	cd $(@D)/.. && $(MAKE)

build-image:
	docker build -t ecmdt-local .

clean:
	rm -f $(COMMANDS)

fresh : | clean clearscr all

clearscr:
	clear
