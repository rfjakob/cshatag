PREFIX ?= /usr/local
GITVERSION := $(shell git describe --dirty)
TARGZ := cshatag_${GITVERSION}_$(shell go env GOOS)-static_$(shell go env GOARCH).tar.gz
GPG_KEY_ID ?= 23A02740

.PHONY: all
all: cshatag README.md

# Always rebuild to make sure GITVERSION is up to date.
.PHONY: cshatag
cshatag:
	CGO_ENABLED=0 go build "-ldflags=-X main.GitVersion=${GITVERSION}"

.PHONY: install
install: cshatag
	@mkdir -v -p ${PREFIX}/bin
	@cp -v cshatag ${PREFIX}/bin
	@mkdir -v -p ${PREFIX}/share/man/man1
	@cp -v cshatag.1 ${PREFIX}/share/man/man1

.PHONY: clean
clean:
	rm -f cshatag README.md

.PHONY: format
format:
	go fmt ./...

README.md: cshatag.1 Makefile README.header.md CHANGELOG.md
	cat README.header.md > README.md
	@echo >> README.md
	@echo '```' >> README.md
	MANWIDTH=80 man ./cshatag.1 >> README.md
	@echo '```' >> README.md
	cat CHANGELOG.md >> README.md

.PHONY: test
test: cshatag
	go vet -all .
	./tests/run_tests.sh

.PHONY: release
release: cshatag cshatag.1
	tar --owner=root --group=root -czf ${TARGZ} cshatag cshatag.1
	gpg -u ${GPG_KEY_ID} --armor --detach-sig ${TARGZ}

# According to "tar tf linux-3.0.tar.gz", the file "linux-3.0/virt/kvm/kvm_main.c" is the
# last one in the tarball. If this one exists, we can be reasonably certain that the tarball
# was extracted completely.
#
# We use /tmp because it is usually a tmpfs, and with a tmpfs the disk speed does not influence
# the measurement.
LINUX_LAST_FILE_EXTRACTED=/tmp/linux-3.0/virt/kvm/kvm_main.c
$(LINUX_LAST_FILE_EXTRACTED):
	wget https://cdn.kernel.org/pub/linux/kernel/v3.0/linux-3.0.tar.gz -O /tmp/linux-3.0.tar.gz
	tar xf /tmp/linux-3.0.tar.gz -C /tmp
	# Initial run is slower because it creates the xattrs.
	# Run it already here.
	./cshatag -q -recursive /tmp/linux-3.0

.PHONY: speed
speed: cshatag $(LINUX_LAST_FILE_EXTRACTED)
	hyperfine "./cshatag -q -recursive /tmp/linux-3.0"
