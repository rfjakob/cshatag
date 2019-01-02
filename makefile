PREFIX ?= /usr/local

cshatag: cshatag.c
	gcc -Wall -Wextra cshatag.c -l crypto -o cshatag
	
install:
	cp -v cshatag ${PREFIX}/bin
	mkdir -v -p ${PREFIX}/share/man/man1 2> /dev/null || true
	cp -v cshatag.1 ${PREFIX}/share/man/man1
	
clean:
	rm -f cshatag
