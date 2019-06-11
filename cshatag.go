package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"github.com/pkg/xattr"
)

const xattrSha256 = "user.shatag.sha256"
const xattrTs = "user.shatag.ts"
const zeroSha256 = "0000000000000000000000000000000000000000000000000000000000000000"

type fileTimestamp struct {
	s  uint64
	ns uint32
}

func (ts *fileTimestamp) prettyPrint() string {
	return fmt.Sprintf("%010d.%09d", ts.s, ts.ns)
}

type fileAttr struct {
	ts     fileTimestamp
	sha256 []byte
}

func (a *fileAttr) prettyPrint() string {
	return fmt.Sprintf("%s %s", string(a.sha256), a.ts.prettyPrint())
}

// getStoredAttr reads the stored extendend attributes from a file. The file
// should look like this:
//
//     $ getfattr -d foo.txt
//     user.shatag.sha256="dc9fe2260fd6748b29532be0ca2750a50f9eca82046b15497f127eba6dda90e8"
//     user.shatag.ts="1560177334.020775051"
func getStoredAttr(f *os.File) (attr fileAttr, err error) {
	attr.sha256 = []byte(zeroSha256)
	val, err := xattr.FGet(f, xattrSha256)
	if err == nil {
		copy(attr.sha256, val)
	}
	val, err = xattr.FGet(f, xattrTs)
	if err == nil {
		parts := strings.SplitN(string(val), ".", 2)
		attr.ts.s, _ = strconv.ParseUint(parts[0], 10, 64)
		if len(parts) > 1 {
			ns64, _ := strconv.ParseUint(parts[1], 10, 32)
			attr.ts.ns = uint32(ns64)
		}
	}
	return attr, nil
}

// getMtime reads the actual modification time of file "f" from disk.
func getMtime(f *os.File) (ts fileTimestamp) {
	fi, err := f.Stat()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	if !fi.Mode().IsRegular() {
		fmt.Println("Error: %s is not a regular file", f.Name())
		os.Exit(3)
	}
	ts.s = uint64(fi.ModTime().Unix())
	ts.ns = uint32(fi.ModTime().Nanosecond())
	return
}

// getActualAttr reads the actual modification time and hashes the file content.
func getActualAttr(f *os.File) (attr fileAttr, err error) {
	attr.sha256 = []byte(zeroSha256)
	attr.ts = getMtime(f)
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
	// Check if the file was modified while we were computing the hash
	ts2 := getMtime(f)
	if attr.ts != ts2 {
		return attr, syscall.EINPROGRESS
	}
	attr.sha256 = []byte(fmt.Sprintf("%x", h.Sum(nil)))
	return attr, nil
}

// storeAttr stores "attr" into extended attributes.
// Should look like this afterwards:
//
//     $ getfattr -d foo.txt
//     user.shatag.sha256="dc9fe2260fd6748b29532be0ca2750a50f9eca82046b15497f127eba6dda90e8"
//     user.shatag.ts="1560177334.020775051"
func storeAttr(f *os.File, attr fileAttr) (err error) {
	if runtime.GOOS == "darwin" {
		// SMB or MacOS bug: when working on an SMB mounted filesystem on a Mac, it seems the call
		// to `fsetxattr` does not update the xattr but removes it instead. So it takes two runs
		// of `cshatag` to update the attribute.
		// To work around this issue, we remove the xattr explicitely before setting it again.
		// https://github.com/rfjakob/cshatag/issues/8
		xattr.FRemove(f, xattrTs)
		xattr.FRemove(f, xattrSha256)
	}
	err = xattr.FSet(f, xattrTs, []byte(attr.ts.prettyPrint()))
	if err != nil {
		return
	}
	err = xattr.FSet(f, xattrSha256, attr.sha256)
	return
}

// printComparison prints something like this:
//
//     stored: faa28bfa6332264571f28b4131b0673f0d55a31a2ccf5c873c435c235647bf76 1560177189.769244818
//     actual: dc9fe2260fd6748b29532be0ca2750a50f9eca82046b15497f127eba6dda90e8 1560177334.020775051
func printComparison(stored fileAttr, actual fileAttr) {
	fmt.Printf(" stored: %s\n actual: %s\n", stored.prettyPrint(), actual.prettyPrint())
}

var stats struct {
	total      int
	errors     int
	inprogress int
	corrupt    int
	outdated   int
	ok         int
}

func checkFile(fn string) {
	stats.total++
	f, err := os.Open(fn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		stats.errors++
		return
	}
	defer f.Close()

	stored, _ := getStoredAttr(f)
	actual, err := getActualAttr(f)
	if err == syscall.EINPROGRESS {
		fmt.Printf("<concurrent modification> %s\n", fn)
		stats.inprogress++
		return
	}
	if stored.ts == actual.ts {
		if bytes.Equal(stored.sha256, actual.sha256) {
			fmt.Printf("<ok> %s\n", fn)
			stats.ok++
			return
		}
		fmt.Fprintf(os.Stderr, "Error: corrupt file %q\n", fn)
		fmt.Printf("<corrupt> %s\n", fn)
		printComparison(stored, actual)
		stats.corrupt++
		return
	}
	fmt.Printf("<outdated> %s\n", fn)
	printComparison(stored, actual)
	err = storeAttr(f, actual)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		stats.errors++
		return
	}
	stats.outdated++
}

func main() {
	myname := os.Args[0]
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s FILE [FILE ...]\n", myname)
		os.Exit(1)
	}
	for _, fn := range os.Args[1:] {
		checkFile(fn)
	}
	if (stats.ok + stats.outdated) == stats.total {
		os.Exit(0)
	}
	if stats.corrupt > 0 {
		os.Exit(5)
	}
	os.Exit(2)
}
