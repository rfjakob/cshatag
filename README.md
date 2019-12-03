[![Build Status](https://travis-ci.org/rfjakob/cshatag.svg?branch=master)](https://travis-ci.org/rfjakob/cshatag)
[![Go Report Card](https://goreportcard.com/badge/github.com/rfjakob/cshatag)](https://goreportcard.com/report/github.com/rfjakob/cshatag)
[Changelog](CHANGELOG.md)

```
CSHATAG(1)                       User Manuals                       CSHATAG(1)

NAME
       cshatag - compiled shatag

SYNOPSIS
       cshatag [OPTIONS] FILE [FILE...]

DESCRIPTION
       cshatag is a minimal and fast re-implementation of shatag
       (  https://bitbucket.org/maugier/shatag  ,  written in python by Maxime
       Augier )
       in a compiled language.

       cshatag is a tool to detect silent data corruption. It writes the mtime
       and  the sha256 checksum of a file into the file's extended attributes.
       The filesystem needs to be mounted with user_xattr enabled for this  to
       work.   When  run  again,  it compares stored mtime and checksum. If it
       finds that the mtime is unchanged but  the  checksum  has  changed,  it
       warns  on  stderr.   In  any case, the status of the file is printed to
       stdout and the stored checksum is updated.

       File statuses that appear on stdout are:
            <outdated>    both mtime and checksum have changed
            <ok>          both checksum and mtime stayed the same
            <timechange>  only mtime has changed, checksum stayed the same
            <corrupt>     mtime stayed the same but checksum changed

       cshatag aims to be format-compatible with shatag and uses the same  ex‐
       tended attributes (see the COMPATIBILITY section).

       cshatag was written in C in 2012 and has been rewritten in Go in 2019.

OPTIONS
       -recursive  recursively process the contents of directories
       -remove     remove cshatag's xattrs from FILE
       -q          quiet mode - don't report <ok> files
       -qq         quiet2 mode - only report <corrupt> files and errors

EXAMPLES
       Check  all regular files in the file tree below the current working di‐
       rectory:
       # cshatag -recursive . > cshatag.log
       Errors like corrupt files will then be printed to stderr  or  grep  for
       "corrupt" in cshatag.log.

       To remove the extended attributes from all files:
       # cshatag -recursive -remove .

RETURN VALUE
       0 Success
       1 Wrong number of arguments
       2 One or more files could not be opened
       3 One or more files is not a regular file
       4 Extended attributes could not be written to one or more files
       5 At least one file was found to be corrupt
       6 More than one type of error occurred

COMPATIBILITY
       cshatag  writes  the  user.shatag.ts field with full integer nanosecond
       precision, while python uses a double for the whole mtime and loses the
       last few digits.

AUTHOR
       Jakob   Unterwurzacher   <jakobunt@gmail.com>,   https://github.com/rf‐
       jakob/cshatag

COPYRIGHT
       Copyright 2012 Jakob Unterwurzacher. MIT License.

SEE ALSO
       shatag(1), sha256sum(1), getfattr(1), setfattr(1)

Linux                              MAY 2012                         CSHATAG(1)
```
