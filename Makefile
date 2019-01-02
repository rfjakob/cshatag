PREFIX ?= /usr/local

.PHONY: all
all: cshatag README

cshatag: cshatag.c
	gcc -Wall -Wextra cshatag.c -l crypto -o cshatag
	
install:
	cp -v cshatag ${PREFIX}/bin
	mkdir -v -p ${PREFIX}/share/man/man1 2> /dev/null || true
	cp -v cshatag.1 ${PREFIX}/share/man/man1
	
clean:
	rm -f cshatag

# Depends on cshatag compilation to make sure the syntax is ok.
format: cshatag
	clang-format -i *.c

README: cshatag.1
	MANWIDTH=80 man ./cshatag.1 > README
