PREFIX ?= /usr/local
GITVERSION := $(shell git describe --dirty)
TARGZ := cshatag_${GITVERSION}_$(shell go env GOOS)-static_$(shell go env GOARCH).tar.gz
GPG_KEY_ID ?= 23A02740

.PHONY: all
all: cshatag README.md

cshatag: *.go Makefile
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

README.md: cshatag.1 Makefile header.md
	cat header.md > README.md
	@echo >> README.md
	@echo '```' >> README.md
	MANWIDTH=80 man ./cshatag.1 >> README.md
	@echo '```' >> README.md

.PHONY: test
test: cshatag
	go vet -all .
	./tests/run_tests.sh

.PHONY: release
release: cshatag cshatag.1
	tar --owner=root --group=root -czf ${TARGZ} cshatag cshatag.1
	gpg -u ${GPG_KEY_ID} --armor --detach-sig ${TARGZ}
