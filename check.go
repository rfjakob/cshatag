package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
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

type fileTimestamp struct {
	s  uint64
	ns uint32
}

func (ts *fileTimestamp) prettyPrint() string {
	if ts == nil {
		return "----------.---------"
	}
	return fmt.Sprintf("%010d.%09d", ts.s, ts.ns)
}

// equalTruncatedTimestamp compares ts and ts2 with 100ns resolution (Linux) or 1s (MacOS).
// Why 100ns? That's what Samba and the Linux SMB client supports.
// Why 1s? That's what the MacOS SMB client supports.
func (ts *fileTimestamp) equalTruncatedTimestamp(ts2 *fileTimestamp) bool {
	if ts == nil || ts2 == nil {
		// an unknown timestamp is never equal to anything
		return false
	}
	if ts.s != ts2.s {
		return false
	}
	if runtime.GOOS == "darwin" {
		// We only look at integer seconds on MacOS, so we are done here.
		return true
	}
	if ts.ns/100 != ts2.ns/100 {
		return false
	}
	return true
}

type fileAttr struct {
	// nil if unknown.
	ts *fileTimestamp
	// sha256 contains the raw sha256 bytes. Length 32.
	// These bytes get converted to hex when stored in
	// the xattr.
	//
	// nil if unknown.
	sha256 []byte
}

func (a *fileAttr) prettyPrint() string {
	var sha256Hex string
	if a.sha256 == nil {
		sha256Hex = strings.Repeat("-", 64)
	} else {
		sha256Hex = hex.EncodeToString(a.sha256)
	}
	return fmt.Sprintf("%s %s", sha256Hex, a.ts.prettyPrint())
}

// getStoredAttr reads the stored extendend attributes from a file. The file
// should look like this:
//
//	$ getfattr -d foo.txt
//	user.shatag.sha256="dc9fe2260fd6748b29532be0ca2750a50f9eca82046b15497f127eba6dda90e8"
//	user.shatag.ts="1560177334.020775051"
func getStoredAttr(f *os.File) (attr fileAttr) {
	val, err := xattr.FGet(f, xattrSha256)
	if err == nil {
		if len(val) >= 64 {
			if len(val) > 64 {
				fmt.Fprintf(os.Stderr, "Warning: user.shatag.sha256 xattr: ignoring trailing garbage (%d bytes)\n", len(val)-64)
				val = val[:64]
			}
			attr.sha256, err = hex.DecodeString(string(val))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: user.shatag.sha256 xattr: hex decode: %s\n", err)
				attr.sha256 = nil
			}
		} else {
			fmt.Fprintf(os.Stderr, "Error: user.shatag.sha256 xattr: incomplete value (%d bytes)\n", len(val))
		}
	}

	val, err = xattr.FGet(f, xattrTs)
	if err == nil {
		parts := strings.SplitN(string(val), ".", 2)
		seconds, err := strconv.ParseUint(parts[0], 10, 64)
		if err != nil {
			return
		}
		var nanoseconds uint64
		if len(parts) > 1 {
			nanoseconds, err = strconv.ParseUint(parts[1], 10, 32)
			if err != nil {
				return
			}
		}
		attr.ts = &fileTimestamp{
			s:  seconds,
			ns: uint32(nanoseconds),
		}
	}
	return
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
	ts, err := getMtime(f)
	if err != nil {
		return attr, err
	}
	attr.ts = &ts

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return attr, err
	}
	// Check if the file was modified while we were computing the hash
	ts2, err := getMtime(f)
	if err != nil {
		return attr, err
	} else if *attr.ts != ts2 {
		return attr, syscall.EINPROGRESS
	}
	attr.sha256 = h.Sum(nil)
	return attr, nil
}

// printComparison prints something like this:
//
//	stored: faa28bfa6332264571f28b4131b0673f0d55a31a2ccf5c873c435c235647bf76 1560177189.769244818
//	actual: dc9fe2260fd6748b29532be0ca2750a50f9eca82046b15497f127eba6dda90e8 1560177334.020775051
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

	stored := getStoredAttr(f)
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

	if stored.ts.equalTruncatedTimestamp(actual.ts) {
		if bytes.Equal(stored.sha256, actual.sha256) {
			if !args.q {
				fmt.Printf("<ok> %s\n", fn)
			}
			stats.ok++
			return
		}
		fixing := " Keeping hash as-is (use -fix to force hash update)."
		if args.fix {
			fixing = " Fixing hash (-fix was passed)."
		}
		fmt.Fprintf(os.Stderr, "Error: corrupt file %q. %s\n", fn, fixing)
		fmt.Printf("<corrupt> %s\n", fn)
		stats.corrupt++
	} else if bytes.Equal(stored.sha256, actual.sha256) {
		if !args.qq {
			fmt.Printf("<timechange> %s\n", fn)
		}
		stats.timechange++
	} else if stored.sha256 == nil && stored.ts == nil {
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

	// Only update the stored attribute if it is not corrupted **OR**
	// if argument '-fix' been given.
	if stored.ts == nil || *stored.ts != *actual.ts || args.fix {
		err = storeAttr(f, actual)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		stats.errorsWritingXattr++
		return
	}
}
