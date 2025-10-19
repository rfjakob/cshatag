package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"runtime/pprof"

	"github.com/charlievieth/fastwalk"
)

// GitVersion is set by the Makefile and contains the version string.
var GitVersion = ""

var args struct {
	remove     bool
	recursive  bool
	q          bool
	qq         bool
	dryrun     bool
	fix        bool
	cpuprofile string
	jobs       int
}

// walkFn is used when `cshatag` is called with the `--recursive` option. It is the function called
// for each file or directory visited whilst traversing the file tree.
func walkFn(path string, info fs.DirEntry, err error) error {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error accessing %q: %v\n", path, err)
		stats.errorsOpening.Add(1)
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
		stats.errorsOpening.Add(1)
	} else if fi.Mode().IsRegular() {
		checkFile(fn)
	} else if fi.IsDir() {
		if args.recursive {
			config := fastwalk.Config{
				NumWorkers: args.jobs,
				Sort:       fastwalk.SortLexical,
			}
			fastwalk.Walk(&config, fn, walkFn)
		} else {
			fmt.Fprintf(os.Stderr, "Error: %q is a directory, did you mean to use the '-recursive' option?\n", fn)
			stats.errorsNotRegular.Add(1)
		}
	} else {
		fmt.Fprintf(os.Stderr, "Error: %q is not a regular file.\n", fn)
		stats.errorsNotRegular.Add(1)
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
	flag.IntVar(&args.jobs, "j", 0, "Number of threads to use. Default: auto")
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

	if stats.decisions.corrupt.Load() > 0 {
		return 5
	}

	totalErrors := stats.errorsOpening.Load() + stats.errorsNotRegular.Load() + stats.errorsWritingXattr.Load() +
		stats.errorsOther.Load()
	if totalErrors > 0 {
		if stats.errorsOpening.Load() == totalErrors {
			return 2
		} else if stats.errorsNotRegular.Load() == totalErrors {
			return 3
		} else if stats.errorsWritingXattr.Load() == totalErrors {
			return 4
		}
		return 6
	}
	if stats.decisions.ok.Load()+
		stats.decisions.outdated.Load()+
		stats.decisions.timechange.Load()+
		stats.decisions.new.Load()+
		stats.removed.Load() == stats.total.Load() {
		return 0
	}
	return 6
}
