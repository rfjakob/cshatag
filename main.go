package main

import (
	"flag"
	"fmt"
	"os"
	"io/fs"
	"runtime/pprof"

	"github.com/charlievieth/fastwalk"
)

// GitVersion is set by the Makefile and contains the version string.
var GitVersion = ""

var stats struct {
	total              int
	errorsNotRegular   int
	errorsOpening      int
	errorsWritingXattr int
	errorsOther        int
	inprogress         int
	removed            int
	decisions          map[decision]int
}

var args struct {
	remove     bool
	recursive  bool
	q          bool
	qq         bool
	dryrun     bool
	fix        bool
	cpuprofile string
}

func init() {
	stats.decisions = make(map[decision]int)
}

// walkFn is used when `cshatag` is called with the `--recursive` option. It is the function called
// for each file or directory visited whilst traversing the file tree.
func walkFn(path string, info fs.DirEntry, err error) error {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error accessing %q: %v\n", path, err)
		stats.errorsOpening++
	} else if info.Type().IsRegular() {
		checkFile(path)
	} else if !info.IsDir() {
		if !args.qq {
			fmt.Printf("<nonregular> %s\n", path)
		}
	}
	return nil
}

// processArg is called for each command-line argument given. For regular files it will call
// `checkFile`. Directories will be processed recursively provided the `--recursive` flag is set.
// Symbolic links are not followed.
func processArg(fn string) {
	fi, err := os.Lstat(fn) // Using Lstat to be consistent with filepath.Walk for symbolic links.
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		stats.errorsOpening++
	} else if fi.Mode().IsRegular() {
		checkFile(fn)
	} else if fi.IsDir() {
		if args.recursive {
			config := fastwalk.Config{
				NumWorkers: 1,
				Sort: fastwalk.SortLexical,
			}
			fastwalk.Walk(&config, fn, walkFn)
		} else {
			fmt.Fprintf(os.Stderr, "Error: %q is a directory, did you mean to use the '-recursive' option?\n", fn)
			stats.errorsNotRegular++
		}
	} else {
		fmt.Fprintf(os.Stderr, "Error: %q is not a regular file.\n", fn)
		stats.errorsNotRegular++
	}
}

func main() {
	// the _main wrapper exists so deferred function can run
	// before os.Exit is called.
	os.Exit(_main())
}

func _main() int {
	const myname = "cshatag"

	if GitVersion == "" {
		GitVersion = "(version unknown)"
	}

	flag.BoolVar(&args.remove, "remove", false, "Remove any previously stored extended attributes.")
	flag.BoolVar(&args.q, "q", false, "quiet: don't print <ok> files")
	flag.BoolVar(&args.qq, "qq", false, "quiet²: Only print <corrupt> files and errors")
	flag.BoolVar(&args.recursive, "recursive", false, "Recursively descend into subdirectories. "+
		"Symbolic links are not followed.")
	flag.BoolVar(&args.dryrun, "dry-run", false, "don't make any changes")
	flag.BoolVar(&args.fix, "fix", false, "fix the stored sha256 on corrupt files")
	flag.StringVar(&args.cpuprofile, "cpuprofile", "", "save cpu profile to specified file")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s %s\n", myname, GitVersion)
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS] FILE [FILE2 ...]\n", myname)
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

	if args.cpuprofile != "" {
		f, err := os.Create(args.cpuprofile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Fatal: %v\n", err)
			os.Exit(1)
		}
		err = pprof.StartCPUProfile(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Fatal: %v\n", err)
			os.Exit(1)
		}
		defer pprof.StopCPUProfile()
	}

	for _, fn := range flag.Args() {
		processArg(fn)
	}

	if stats.decisions[decisionCorrupt] > 0 {
		return 5
	}

	totalErrors := stats.errorsOpening + stats.errorsNotRegular + stats.errorsWritingXattr +
		stats.errorsOther
	if totalErrors > 0 {
		if stats.errorsOpening == totalErrors {
			return 2
		} else if stats.errorsNotRegular == totalErrors {
			return 3
		} else if stats.errorsWritingXattr == totalErrors {
			return 4
		}
		return 6
	}
	if (stats.decisions[decisionOk]+
		stats.decisions[decisionOutdated]+
		stats.decisions[decisionTimechange]+
		stats.decisions[decisionNew])+
		stats.removed == stats.total {
		return 0
	}
	return 6
}
