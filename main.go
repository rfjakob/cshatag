package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

var GitVersion = ""

var stats struct {
	total      int
	errors     int
	inprogress int
	corrupt    int
	timechange int
	outdated   int
	ok         int
}

var args struct {
	remove    bool
	recursive bool
	q         bool
	qq        bool
}

// walkFn is used when `cshatag` is called with the `--recursive` option. It is the function called
// for each file or directory visited whilst traversing the file tree.
func walkFn(path string, info os.FileInfo, err error) error {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error accessing %q: %v\n", path, err)
		stats.errors++
	} else if info.Mode().IsRegular() {
		checkFile(path)
	} else if !info.IsDir() {
		fmt.Fprintf(os.Stderr, "Error: %q is not a regular file.\n", path)
		stats.errors++
	}
	return nil
}

// processDir will read the contents of the directory named "fn" and for each regular file found
// within, call `checkFile`. This function is used when a directory is passed as a command-line
// argument but the `--recursive` flag is not set.
func processDir(fn string) {
	files, err := ioutil.ReadDir(fn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error processing directory %q: %v\n", fn, err)
		stats.errors++
		return
	}
	for _, file := range files {
		if !file.IsDir() {
			if file.Mode().IsRegular() {
				checkFile(filepath.Join(fn, file.Name()))
			} else {
				fmt.Fprintf(os.Stderr, "Error: %q is not a regular file.\n", file.Name())
				stats.errors++
			}
		}
	}
}

// processArg is called for each command-line argument given. For regular files it will call
// `checkFile`. For directories the behaviour depends on whether the `--recursive` flag is set.
// Symbolic links are not followed.
func processArg(fn string) {
	fi, err := os.Lstat(fn) // Using Lstat to be consistent with filepath.Walk for symbolic links.
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		stats.errors++
	} else if fi.Mode().IsRegular() {
		checkFile(fn)
	} else if fi.IsDir() {
		if args.recursive {
			filepath.Walk(fn, walkFn)
		} else {
			processDir(fn)
		}
	} else {
		fmt.Fprintf(os.Stderr, "Error: %q is not a regular file.\n", fn)
		stats.errors++
	}
}

func main() {
	const myname = "cshatag"

	if GitVersion == "" {
		GitVersion = "(version unknown)"
	}

	flag.BoolVar(&args.remove, "remove", false, "Remove any previously stored extended attributes.")
	flag.BoolVar(&args.q, "q", false, "quiet: don't print <ok> files")
	flag.BoolVar(&args.qq, "qq", false, "quiet²: Only print <corrupt> files and errors")
	flag.BoolVar(&args.recursive, "recursive", false, "Recursively descend into subdirectories. "+
		"Symbolic links are not followed.")
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
	if args.qq {
		// quiet2 implies quiet
		args.q = true
	}

	for _, fn := range flag.Args() {
		processArg(fn)
	}

	if stats.corrupt > 0 {
		os.Exit(3)
	}
	if stats.errors > 0 {
		os.Exit(2)
	}
	if (stats.ok + stats.outdated + stats.timechange) == stats.total {
		os.Exit(0)
	}
	os.Exit(2)
}
