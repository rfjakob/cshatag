[![CI](https://github.com/rfjakob/cshatag/actions/workflows/ci.yml/badge.svg)](https://github.com/rfjakob/cshatag/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/rfjakob/cshatag)](https://goreportcard.com/report/github.com/rfjakob/cshatag)
•
[View Changelog](CHANGELOG.md)
•
[Download Binary Releases](https://github.com/rfjakob/cshatag/releases)

cshatag is a tool to detect silent data corruption. It is meant to run periodically
and stores the SHA256 of each file as an extended attribute.
See the [Man Page](#man-page) below for details.

Compile Yourself
----------------
```
$ git clone https://github.com/rfjakob/cshatag.git
$ cd cshatag
$ make
```

Man Page
--------
