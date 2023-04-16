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

func zeroFileTimeStamp() fileTimestamp {
	return fileTimestamp{
		s:  uint64(0),
		ns: uint32(0),
	}
}

func (ts *fileTimestamp) prettyPrint() string {
	return fmt.Sprintf("%010d.%09d", ts.s, ts.ns)
}

// equalTruncatedTimestamp compares ts and ts2 with 100ns resolution (Linux) or 1s (MacOS).
// Why 100ns? That's what Samba and the Linux SMB client supports.
// Why 1s? That's what the MacOS SMB client supports.
func (ts *fileTimestamp) equalTruncatedTimestamp(ts2 *fileTimestamp) bool {
	if ts.s != ts2.s {
		return false
	}
	// We only look at integer seconds on MacOS, so we are done here.
	if runtime.GOOS == "darwin" {
		return true
	}
	if ts.ns/100 != ts2.ns/100 {
		return false
	}
	return true
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
func getMtime(f *os.File) (ts fileTimestamp, err error) {
	fi, err := f.Stat()
	if err != nil {
		return
	}
	ts.s = uint64(fi.ModTime().Unix())
	ts.ns = uint32(fi.ModTime().Nanosecond())
	return
}

// getActualAttr reads the actual modification time and hashes the file content.
func getActualAttr(f *os.File) (attr fileAttr, err error) {
	attr.sha256 = []byte(zeroSha256)
	attr.ts, err = getMtime(f)
	if err != nil {
		return attr, err
	}
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return attr, err
	}
	// Check if the file was modified while we were computing the hash
	ts2, err := getMtime(f)
	if err != nil {
		return attr, err
	} else if attr.ts != ts2 {
		return attr, syscall.EINPROGRESS
	}
	attr.sha256 = []byte(fmt.Sprintf("%x", h.Sum(nil)))
	return attr, nil
}

// printComparison prints something like this:
//
//     stored: faa28bfa6332264571f28b4131b0673f0d55a31a2ccf5c873c435c235647bf76 1560177189.769244818
//     actual: dc9fe2260fd6748b29532be0ca2750a50f9eca82046b15497f127eba6dda90e8 1560177334.020775051
func printComparison(stored fileAttr, actual fileAttr) {
	fmt.Printf(" stored: %s\n actual: %s\n", stored.prettyPrint(), actual.prettyPrint())
}

func checkFile(fn string) {
	stats.total++
	f, err := os.Open(fn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		stats.errorsOpening++
		return
	}
	defer f.Close()

	if args.remove {
		if err = removeAttr(f); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			stats.errorsOther++
			return
		}
		if !args.q {
			fmt.Printf("<removed xattr> %s\n", fn)
		}
		stats.ok++
		return
	}

	stored, _ := getStoredAttr(f)
	actual, err := getActualAttr(f)
	if err == syscall.EINPROGRESS {
		if !args.qq {
			fmt.Printf("<concurrent modification> %s\n", fn)
		}
		stats.inprogress++
		return
	} else if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		stats.errorsOther++
		return
	}

	if stored.ts.equalTruncatedTimestamp(&actual.ts) {
		if bytes.Equal(stored.sha256, actual.sha256) {
			if !args.q {
				fmt.Printf("<ok> %s\n", fn)
			}
			stats.ok++
			return
		}
		fmt.Fprintf(os.Stderr, "Error: corrupt file %q\n", fn)
		fmt.Printf("<corrupt> %s\n", fn)
		stats.corrupt++
	} else if bytes.Equal(stored.sha256, actual.sha256) {
		if !args.qq {
			fmt.Printf("<timechange> %s\n", fn)
		}
		stats.timechange++
	} else if bytes.Equal(stored.sha256, []byte(zeroSha256)) && (stored.ts == zeroFileTimeStamp()) {
		// no metadata indicates a 'new' file
		if !args.qq {
			fmt.Printf("<new> %s\n", fn)
		}
		stats.newfile++
	} else {
		// timestamp is outdated
		if !args.qq {
			fmt.Printf("<outdated> %s\n", fn)
		}
		stats.outdated++
	}
	if !args.qq {
		printComparison(stored, actual)
	}
	err = storeAttr(f, actual)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		stats.errorsWritingXattr++
		return
	}
}
