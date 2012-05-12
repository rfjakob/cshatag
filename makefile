cshatag:
	gcc cshatag.c -l crypto -o cshatag
	
install:
	cp -v cshatag /usr/local/bin
	mkdir -v /usr/local/share/man/man1 2> /dev/null || true
	cp -v cshatag.1 /usr/local/share/man/man1
	
clean:
	rm -f cshatag