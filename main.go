package main

import (
	"flag"
	"fmt"
	"os"
)

var GitVersion = ""
var removeFlag bool

var stats struct {
	total      int
	errors     int
	inprogress int
	corrupt    int
	timechange int
	outdated   int
	ok         int
}

func main() {
	const myname = "cshatag"

	if GitVersion == "" {
		GitVersion = "(version unknown)"
	}

	flag.BoolVar(&removeFlag, "remove", false, "Remove any previously stored extended attributes.")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s %s\n", myname, GitVersion)
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTION] FILE [FILE ...]\n", myname)
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		os.Exit(1)
	}
	flag.Parse()

	if flag.NArg() == 0 {
		flag.Usage()
	}

	for _, fn := range flag.Args() {
		checkFile(fn)
	}
	if (stats.ok + stats.outdated + stats.timechange) == stats.total {
		os.Exit(0)
	}
	if stats.corrupt > 0 {
		os.Exit(5)
	}
	os.Exit(2)
}
