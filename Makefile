PREFIX ?= /usr/local

.PHONY: all
all: cshatag README.md

cshatag: cshatag.c Makefile
	gcc -Wall -Wextra cshatag.c -l crypto -o cshatag

.PHONY: install
install: cshatag
	@mkdir -v -p ${PREFIX}/bin
	@cp -v cshatag ${PREFIX}/bin
	@mkdir -v -p ${PREFIX}/share/man/man1
	@cp -v cshatag.1 ${PREFIX}/share/man/man1

.PHONY: clean
clean:
	rm -f cshatag README.md

# Depends on cshatag compilation to make sure the syntax is ok.
.PHONY: format
format: cshatag
	clang-format -i *.c

README.md: cshatag.1 Makefile
	@echo '[![Build Status](https://travis-ci.org/rfjakob/cshatag.svg?branch=master)](https://travis-ci.org/rfjakob/cshatag)' > README.md
	@echo '[![Total alerts](https://img.shields.io/lgtm/alerts/g/rfjakob/cshatag.svg?logo=lgtm&logoWidth=18)](https://lgtm.com/projects/g/rfjakob/cshatag/alerts/)' >> README.md
	@echo '[![Language grade: C/C++](https://img.shields.io/lgtm/grade/cpp/g/rfjakob/cshatag.svg?logo=lgtm&logoWidth=18)](https://lgtm.com/projects/g/rfjakob/cshatag/context:cpp)' >> README.md
	@echo >> README.md
	@echo '```' >> README.md
	MANWIDTH=80 man ./cshatag.1 >> README.md
	@echo '```' >> README.md

.PHONY: test
test: cshatag
	./tests/run_tests.sh
