package main

// This file has all functions that actually change something on disk

import (
	"errors"
	"os"
	"runtime"

	"github.com/pkg/xattr"
)

// storeAttr stores "attr" into extended attributes.
// Should look like this afterwards:
//
//	$ getfattr -d foo.txt
//	user.shatag.sha256="dc9fe2260fd6748b29532be0ca2750a50f9eca82046b15497f127eba6dda90e8"
//	user.shatag.ts="1560177334.020775051"
func storeAttr(f *os.File, attr fileAttr) (err error) {
	if args.dryrun {
		return nil
	}
	if runtime.GOOS == "darwin" {
		// SMB or MacOS bug: when working on an SMB mounted filesystem on a Mac, it seems the call
		// to `fsetxattr` does not update the xattr but removes it instead. So it takes two runs
		// of `cshatag` to update the attribute.
		// To work around this issue, we remove the xattr explicitly before setting it again.
		// https://github.com/rfjakob/cshatag/issues/8
		removeAttr(f)
	}
	err = xattr.FSet(f, xattrTs, []byte(attr.ts.prettyPrint()))
	if err != nil {
		return
	}
	err = xattr.FSet(f, xattrSha256, attr.sha256)
	return
}

// removeAttr removes any previously stored extended attributes. Returns an error
// if removal of either the timestamp or checksum xattrs fails.
func removeAttr(f *os.File) error {
	if args.dryrun {
		return nil
	}
	err1 := xattr.FRemove(f, xattrTs)
	err2 := xattr.FRemove(f, xattrSha256)
	if err1 != nil && err2 != nil {
		return errors.New(err1.Error() + "  " + err2.Error())
	}
	if err1 != nil {
		return err1
	}
	if err2 != nil {
		return err2
	}
	return nil
}
