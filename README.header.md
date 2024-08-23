[![CI](https://github.com/rfjakob/cshatag/actions/workflows/ci.yml/badge.svg)](https://github.com/rfjakob/cshatag/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/rfjakob/cshatag)](https://goreportcard.com/report/github.com/rfjakob/cshatag)
•
[View Changelog](#Changelog)
•
[Download Binary Releases](https://github.com/rfjakob/cshatag/releases)

cshatag is a tool to detect silent data corruption. It is meant to run periodically
and stores the SHA256 of each file as an extended attribute. The project started
as a minimal and fast reimplementation of [shatag](https://github.com/maugier/shatag),
written in Python by Maxime Augier.

See the [Man Page](#man-page) further down this page for details.

Similar Tools
-------------

Checksums stored in extended attributes for each file
* https://github.com/maugier/shatag (the original shatag tool, written in Python)

Checksums stored in single central database
* https://github.com/ambv/bitrot
* https://sourceforge.net/p/yabitrot/code/ci/master/tree/

Checksums stored in one database per directory
* https://github.com/laktak/chkbit-py

Download
--------
Static amd64 binary that should work on all Linux distros:
https://github.com/rfjakob/cshatag/releases

Distro Packages
---------------
[![Packaging status](https://repology.org/badge/vertical-allrepos/cshatag.svg)](https://repology.org/project/cshatag/versions)

Compile Yourself
----------------
Needs git, Go and make.

```
$ git clone https://github.com/rfjakob/cshatag.git
$ cd cshatag
$ make
```

Man Page
--------
