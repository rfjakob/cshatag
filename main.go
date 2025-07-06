package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"sync/atomic"
	"syscall"
)

// GitVersion is set by the Makefile and contains the version string.
var GitVersion = ""

var stats struct {
	total              uint32
	errorsNotRegular   uint32
	errorsOpening      uint32
	errorsWritingXattr uint32
	errorsOther        uint32
	inprogress         uint32
	corrupt            uint32
	timechange         uint32
	outdated           uint32
	newfile            uint32
	ok                 uint32
}

var args struct {
	remove    bool
	recursive bool
	q         bool
	qq        bool
	dryrun    bool
	fix       bool
	jobs      int
}

type Queue chan string

var queue Queue

var ctx context.Context
var cancel context.CancelFunc

// walkFn is used when `cshatag` is called with the `--recursive` option. It is the function called
// for each file or directory visited whilst traversing the file tree.
func walkFn(path string, info os.FileInfo, err error) error {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error accessing %q: %v\n", path, err)
		atomic.AddUint32(&stats.errorsOpening, 1)
	} else if info.Mode().IsRegular() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			queueFile(path)
		}
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
		atomic.AddUint32(&stats.errorsOpening, 1)
	} else if fi.Mode().IsRegular() {
		queueFile(fn)
	} else if fi.IsDir() {
		if args.recursive {
			filepath.Walk(fn, walkFn)
		} else {
			fmt.Fprintf(os.Stderr, "Error: %q is a directory, did you mean to use the '-recursive' option?\n", fn)
			atomic.AddUint32(&stats.errorsNotRegular, 1)
		}
	} else {
		fmt.Fprintf(os.Stderr, "Error: %q is not a regular file.\n", fn)
		atomic.AddUint32(&stats.errorsNotRegular, 1)
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
	flag.BoolVar(&args.dryrun, "dry-run", false, "don't make any changes")
	flag.BoolVar(&args.fix, "fix", false, "fix the stored sha256 on corrupt files")
	flag.IntVar(&args.jobs, "jobs", 1, "Number of parallel jobs to run.")
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

	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handler
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start signal watcher goroutine
	go func() {
		sig := <-sigChan
		fmt.Fprintf(os.Stderr, "\nReceived signal: %s\n", sig)
		cancel()
	}()

	queue = make(chan string, args.jobs)
	var wg sync.WaitGroup

	for i := 1; i <= args.jobs; i++ {
		wg.Add(1)
		go worker(ctx, &wg)
	}

	for _, fn := range flag.Args() {
		processArg(fn)
	}

	close(queue)

	wg.Wait()

	if stats.corrupt > 0 {
		os.Exit(5)
	}

	totalErrors := stats.errorsOpening + stats.errorsNotRegular + stats.errorsWritingXattr +
		stats.errorsOther
	if totalErrors > 0 {
		if stats.errorsOpening == totalErrors {
			os.Exit(2)
		} else if stats.errorsNotRegular == totalErrors {
			os.Exit(3)
		} else if stats.errorsWritingXattr == totalErrors {
			os.Exit(4)
		}
		os.Exit(6)
	}
	if (stats.ok + stats.outdated + stats.timechange + stats.newfile) == stats.total {
		os.Exit(0)
	}
	os.Exit(6)
}
