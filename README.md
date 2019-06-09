[![Build Status](https://travis-ci.org/rfjakob/cshatag.svg?branch=master)](https://travis-ci.org/rfjakob/cshatag)
[![Total alerts](https://img.shields.io/lgtm/alerts/g/rfjakob/cshatag.svg?logo=lgtm&logoWidth=18)](https://lgtm.com/projects/g/rfjakob/cshatag/alerts/)
[![Language grade: C/C++](https://img.shields.io/lgtm/grade/cpp/g/rfjakob/cshatag.svg?logo=lgtm&logoWidth=18)](https://lgtm.com/projects/g/rfjakob/cshatag/context:cpp)

```
CSHATAG(1)                       User Manuals                       CSHATAG(1)

NAME
       cshatag - shatag in C

SYNOPSIS
       cshatag FILE

DESCRIPTION
       cshatag is a minimal re-implementation in C of shatag
       (  https://bitbucket.org/maugier/shatag  ,  written in python by Maxime
       Augier ).

       cshatag is a tool to detect silent data corruption. It writes the mtime
       and  the sha256 checksum of a file into the file's extended attributes.
       The filesystem needs to be mounted with user_xattr enabled for this  to
       work.   When  run  again,  it compares stored mtime and checksum. If it
       finds that the mtime is unchanged but  the  checksum  has  changed,  it
       warns  on  stderr.   In  any case, the status of the file is printed to
       stdout and the stored checksum is updated.

       File statuses that appear on stdout are:
            outdated    mtime has changed
            ok          mtime has not changed, checksum is correct
            corrupt     mtime has not changed, checksum is wrong

       cshatag aims to be format-compatible with  shatag  and  uses  the  same
       extended attributes (see the COMPATIBILITY section).

EXAMPLES
       Typically, cshatag will be called from find:
       # find / -xdev -type f -exec cshatag {} \; > cshatag.log
       Errors  like  corrupt  files will then be printed to stderr or grep for
       "corrupt" in cshatag.log.

       To remove the extended attributes from all files:
       # find / -xdev -type f -exec setfattr -x  user.shatag.ts  {}  \;  -exec
       setfattr -x user.shatag.sha256 {} \;

RETURN VALUE
       0 Success
       1 Invalid command-line
       8 Corrupt file(s) found
       16 Error(s) encountered during processing
       24 Corrupt file(s) found AND error(s) encountered (= 8 + 16)

COMPATIBILITY
       cshatag  writes  the  user.shatag.ts field with full integer nanosecond
       precision, while python uses a double for the whole mtime and loses the
       last few digits.

AUTHOR
       Jakob Unterwurzacher <jakobunt@gmail.com>

COPYRIGHT
       Copyright 2012 Jakob Unterwurzacher. License GPLv2+.

SEE ALSO
       shatag(1), sha256sum(1), getfattr(1), setfattr(1)

Linux                              MAY 2012                         CSHATAG(1)
```
